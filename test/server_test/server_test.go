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

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T, client api.LogClient,
		config *server.Config,
	){
		"produce/consume a message to/from the log succeeds": testProduceConsume,
		"consume past boundary fails":                        testConsumePastBoundary,
		"produce/consume stream succeeds":                    testProduceConsumeStream,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

func setupTest(t *testing.T, fn func(*server.Config)) (
	client api.LogClient,
	cfg *server.Config,
	teardown func(),
) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile: config.CAFile,
	})
	clientCreds := credentials.NewTLS(clientTLSConfig)

	//clientOpts := []grpc.DialOption{grpc.WithInsecure()}
	//conn, err := grpc.Dial(listener.Addr().String(), clientOpts...)
	conn, err := grpc.Dial(listener.Addr().String(),
		grpc.WithTransportCredentials(clientCreds),
	)
	require.NoError(t, err)

	client = api.NewLogClient(conn)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile:       config.CAFile,
		CertFile:     config.ServerCertFile,
		KeyFile:      config.ServerKeyFile,
		ServerAdress: listener.Addr().String(),
	})
	require.NoError(t, err)
	serverCreds := credentials.NewTLS(serverTLSConfig)

	dir, err := ioutil.TempDir("", "server-test")
	require.NoError(t, err)

	log, err := lg.NewLog(dir, lg.Config{})
	require.NoError(t, err)

	err = log.NewSegment(uint64(0))
	require.NoError(t, err)

	cfg = &server.Config{
		CommitLog: log,
	}
	if fn != nil {
		fn(cfg)
	}

	srv, err := server.NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		srv.Serve(listener)
	}()

	return client, cfg, func() {
		srv.Stop()
		conn.Close()
		listener.Close()
		//log.Remove()
	}
}

func testProduceConsume(t *testing.T, client api.LogClient, cfg *server.Config) {
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
	client api.LogClient,
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
	client api.LogClient,
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
