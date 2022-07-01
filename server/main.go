package main

import (
	"context"
	"fmt"
	proto "github.com/krodz/rpcgame/protobuf"
	"github.com/krodz/rpcgame/server/logic"
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
	l := log.New(os.Stdout, "", 0)
	// setup OTEL
	f, err := os.Create("traces.txt")
	if err != nil {
		l.Fatal(err)
	}
	defer f.Close()

	exp, err := newExporter(f)
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

	// SETUP GRPC SERVER
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 8080))
	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()), // grpc otel
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor())) // grpc otel

	proto.RegisterPlayerServiceServer(grpcServer, logic.NewServer())
	log.Println("listening to post 8080")
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
