package main

import (
	"context"
	proto "github.com/krodz/rpcgame/protobuf"
	"github.com/krodz/rpcgame/server/logic"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"google.golang.org/grpc"
	"io"
	"log"
	"net"
	"os"
)

func main() {
	ctx := context.Background()
	// setup OTEL
	f, err := os.Create("traces.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	exp, err := newExporter(f)
	if err != nil {
		log.Fatal(err)
	}

	env := os.Getenv("OTEL_EXPORTER_JAEGER_ENDPOINT")
	log.Printf("ENV KEY: %s", env)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
		trace.WithResource(newResource()),
	)
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// SETUP GRPC SERVER
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),   // grpc otel
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor())) // grpc otel

	// connecting to DB
	dbURL := os.Getenv("RPCGAME_MONGODB_URL")
	if dbURL == "" {
		panic("missing env RPCGAME_MONGODB_URL")
	}

	opt := options.Client()
	opt.Monitor = otelmongo.NewMonitor()

	client, err := mongo.Connect(ctx, opt.ApplyURI(dbURL))
	defer client.Disconnect(ctx)
	if err != nil {
		panic(err)
	}

	proto.RegisterPlayerServiceServer(grpcServer, logic.NewServer(client))
	log.Println("listening to port 8080")
	grpcServer.Serve(lis)
}

func newExporter(w io.Writer) (trace.SpanExporter, error) {
	var opts []jaeger.AgentEndpointOption
	return jaeger.New(jaeger.WithAgentEndpoint(opts...))
}

func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("player"),
			semconv.ServiceVersionKey.String("v0.0.1"),
			attribute.String("environment", "demo"),
		),
	)
	return r
}
