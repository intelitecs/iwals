package main

import (
	"context"
	"flag"
	"fmt"
	api "iwals/api/v1"
	"log"

	"google.golang.org/grpc"
)

func main() {
	//srv := server.NewHTTPServer(":8090")
	//log.Fatal(srv.ListenAndServe())

	addr := flag.String("addr", ":8400", "service address")
	flag.Parse()
	conn, err := grpc.Dial(*addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	client := api.NewLogClient(conn)
	ctx := context.Background()
	res, err := client.GetServers(ctx, &api.GetServersRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("servers:")
	for _, server := range res.Servers {
		fmt.Printf("\t- %v\n", server)
	}

}
