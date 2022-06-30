package logic

import (
	"context"
	"errors"
	"github.com/krodz/gomicrotest/proto"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/metadata"
	"log"
	"time"
)

type playerServer struct {
	proto.UnimplementedPlayerServiceServer
}

func NewServer() proto.PlayerServiceServer {
	return &playerServer{}
}

func (s *playerServer) Introduce(ctx context.Context, noInput *proto.NoInput) (*proto.Player, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		log.Println(md)
	}

	ctx, span := otel.Tracer("PlayerServer").Start(ctx, "Player.Introduce")
	defer span.End()

	defer func(t time.Time) {
		log.Printf("Server.Introduce: Execution Time %v", time.Since(t))
	}(time.Now())
	return s.newPlayer(ctx, "krodz"), nil
}

func (s *playerServer) newPlayer(ctx context.Context, name string) *proto.Player {
	_, span := otel.Tracer("PlayerServer").Start(ctx, "PlayerServer.newPlayer")
	defer span.End()
	return &proto.Player{Name: name, Health: 12}
}

func (s *playerServer) GetInventory(p *proto.Player, stream proto.PlayerService_GetInventoryServer) error {
	defer func(t time.Time) {
		log.Printf("Server.GetInventory: Execution Time %v", time.Since(t))
	}(time.Now())
	if p == nil || p.Health == 0 {
		return errors.New("a dead player owns nothing")
	}

	items := []proto.Item{
		{
			Name:        "red potion",
			Description: "heal for 10 hp",
			Quantity:    10,
		},
		{
			Name:        "blue potion",
			Description: "increase mana for 10 mp",
			Quantity:    5,
		},
	}

	for _, item := range items {
		if err := stream.Send(&item); err != nil {
			return err
		}
	}
	return nil
}
