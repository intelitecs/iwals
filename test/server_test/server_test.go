package server_test

import (
	"context"
	"io/ioutil"
	"net"
	"testing"

	api "iwals/api/v1"
	lg "iwals/internal/log"
	server "iwals/internal/server"

	"iwals/internal/config"

	"iwals/internal/security/authorization"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func newClient(certPath, keyPath string, serverListener net.Listener, t *testing.T) (*grpc.ClientConn, api.LogClient, []grpc.DialOption) {
	tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile:   config.CAFile,
		CertFile: certPath,
		KeyFile:  keyPath,
		Server:   false,
	})
	require.NoError(t, err)

	clientCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(clientCreds)}
	conn, err := grpc.Dial(serverListener.Addr().String(),
		opts...,
	)
	require.NoError(t, err)

	client := api.NewLogClient(conn)

	return conn, client, opts
}

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		rootClient,
		nobodyClient api.LogClient,
		confg *server.Config,
	){
		"produce/consume a message to/from the log succeeds": testProduceConsume,
		"consume past boundary fails":                        testConsumePastBoundary,
		"produce/consume stream succeeds":                    testProduceConsumeStream,
		"unauthorized fails":                                 testUnAuthorized,
	} {
		t.Run(scenario, func(t *testing.T) {
			rootClient, nobodyClient, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, rootClient, nobodyClient, config)
		})
	}
}

func setupTest(t *testing.T, fn func(*server.Config)) (
	rootClient api.LogClient,
	nobodyClient api.LogClient,
	cfg *server.Config,
	teardown func(),
) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// server's tls config
	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile:       config.CAFile,
		CertFile:     config.ServerCertFile,
		KeyFile:      config.ServerKeyFile,
		ServerAdress: listener.Addr().String(),
		Server:       true,
	})
	require.NoError(t, err)
	// server's credentials
	serverCreds := credentials.NewTLS(serverTLSConfig)

	// clients
	var rootConn *grpc.ClientConn
	rootConn, rootClient, _ = newClient(config.RootClientCertFile, config.RootClientKeyFile, listener, t)
	var nobodyConn *grpc.ClientConn
	nobodyConn, nobodyClient, _ = newClient(config.NobodyClientCertFile, config.NobodyClientKeyFile, listener, t)

	dir, err := ioutil.TempDir("", "server-test")
	require.NoError(t, err)

	log, err := lg.NewLog(dir, lg.Config{})
	require.NoError(t, err)

	err = log.NewSegment(uint64(0))
	require.NoError(t, err)

	// authorization
	authorizer := authorization.NewAuthorizer(config.ACLModelFile, config.ACLPolicyFile)

	// server's config
	cfg = &server.Config{
		CommitLog:  log,
		Authorizer: authorizer,
	}
	if fn != nil {
		fn(cfg)
	}
	// grpc server
	srv, err := server.NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		srv.Serve(listener)
	}()

	return rootClient, nobodyClient, cfg, func() {
		srv.Stop()
		rootConn.Close()
		nobodyConn.Close()
		listener.Close()
		//log.Remove()
	}
}

func testProduceConsume(t *testing.T, client, _ api.LogClient, cfg *server.Config) {
	ctx := context.Background()
	want := &api.Record{
		Value: []byte("hello world"),
	}

	produce, err := client.Produce(ctx, &api.ProduceRequest{
		Record: want,
	},
	)
	require.NoError(t, err)

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset,
	})
	require.NoError(t, err)
	require.Equal(t, want.Value, consume.Record.Value)
	require.Equal(t, want.Offset, consume.Record.Offset)
}

func testConsumePastBoundary(
	t *testing.T,
	client, _ api.LogClient,
	config *server.Config,
) {
	ctx := context.Background()

	produce, err := client.Produce(ctx, &api.ProduceRequest{
		Record: &api.Record{
			Value: []byte("hello world"),
		},
	})
	require.NoError(t, err)
	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: produce.Offset + 1,
	})
	if consume != nil {
		t.Fatal("consume not nil")
	}
	got := grpc.Code(err)
	want := grpc.Code(api.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	if got != want {
		t.Fatalf("got err: %v, want: %v", got, want)
	}
}

func testProduceConsumeStream(
	t *testing.T,
	client, _ api.LogClient,
	cfg *server.Config,
) {
	ctx := context.Background()
	records := []*api.Record{
		{
			Value:  []byte("first message"),
			Offset: 0,
		},
		{
			Value:  []byte("secod message"),
			Offset: 1,
		}}
	{
		stream, err := client.ProduceStream(ctx)
		require.NoError(t, err)

		for offset, record := range records {
			err := stream.Send(&api.ProduceRequest{
				Record: record,
			})
			require.NoError(t, err)
			res, err := stream.Recv()
			require.NoError(t, err)
			if res.Offset != uint64(offset) {
				t.Fatalf("got %v, want:%v", res.Offset, offset)
			}
		}
	}
	{
		stream, err := client.ConsumeStream(ctx, &api.ConsumeRequest{
			Offset: 0,
		})
		require.NoError(t, err)

		for i, record := range records {
			res, err := stream.Recv()
			require.NoError(t, err)
			require.Equal(t, res.Record, &api.Record{
				Value:  record.Value,
				Offset: uint64(i),
			})
		}
	}

}

func testUnAuthorized(t *testing.T, _, client api.LogClient, cfg *server.Config) {
	ctx := context.Background()
	produce, err := client.Produce(ctx, &api.ProduceRequest{
		Record: &api.Record{
			Value: []byte("hello world"),
		},
	},
	)
	if produce != nil {
		t.Fatalf("response should be nil")
	}
	gotCode, wantCode := status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d ", gotCode, wantCode)
	}

	consume, err := client.Consume(ctx, &api.ConsumeRequest{
		Offset: 0,
	})
	if consume != nil {
		t.Fatalf("consume response should be nil")
	}
	gotCode, wantCode = status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d ", gotCode, wantCode)
	}
}
