package main

import (
	"context"
	"github.com/krodz/gomicrotest/proto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"time"
)

func main() {
	md := metadata.Pairs(
		"timestamp", time.Now().Format(time.StampNano),
		"caller", "krodz",
	)
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	//setup connection
	conn, err := grpc.Dial("localhost:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()))
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	// create player service client
	client := proto.NewPlayerServiceClient(conn)

	// call logic
	logic(ctx, client)
}

func logic(ctx context.Context, client proto.PlayerServiceClient) {

	p, err := client.Introduce(ctx, &proto.NoInput{})

	if err != nil {
		panic(err)
	}
	log.Printf("Introduce player %s with %d health", p.Name, p.Health)

	log.Println("Getting inventory...")
	defer func(t time.Time) {
		log.Printf("Execution Time GetInventory %v", time.Since(t))
	}(time.Now())
	stream, err := client.GetInventory(ctx, p)
	if err != nil {
		panic(err)
	}
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.GetInventory(_) = _, %v", client, err)
		}
		log.Printf("Item %s Description %s Quantity %d \n", item.Name, item.Description, item.Quantity)
	}
}
