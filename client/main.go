package main

import (
	"context"
	"github.com/krodz/gomicrotest/proto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	l := log.New(os.Stdout, "", 0)

	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"creater", "krodz",
	)

	exp, err := newExporter()
	if err != nil {
		l.Fatal(err)
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(newResource()),
	)

	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			l.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

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
	ctx, span := otel.Tracer("PlayerServer").Start(ctx, "client.logic")
	defer span.End()
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

func newExporter() (trace.SpanExporter, error) {
	var opts []jaeger.AgentEndpointOption
	return jaeger.New(jaeger.WithAgentEndpoint(opts...))
}

func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("client"),
			semconv.ServiceVersionKey.String("v0.0.1"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
