package logic

import (
	"context"
	"errors"
	"fmt"
	proto "github.com/krodz/rpcgame/protobuf"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc/metadata"
	"log"
	"time"
)

type playerServer struct {
	proto.UnimplementedPlayerServiceServer
	mongoClient *mongo.Client
}

func NewServer(client *mongo.Client) proto.PlayerServiceServer {
	return &playerServer{
		mongoClient: client,
	}
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
	return s.newPlayer(ctx, "krodz", "test"), nil
}

func (s *playerServer) CreateNewPlayer(ctx context.Context, req *proto.CreatePlayerRequest) (*proto.Player, error) {
	ctx, span := otel.Tracer("PlayerServer").Start(ctx, "PlayerServer.CreateNewPlayer")
	defer span.End()

	p := s.getPlayerByName(ctx, req.Name)
	if p != nil {
		return nil, fmt.Errorf("player name already used")
	}

	p = s.newPlayer(ctx, req.Name, req.Type)

	doc, err := bson.Marshal(p)
	if err != nil {
		return nil, err
	}
	coll := s.getPlayerCollection(ctx)
	results, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	log.Printf("Inserter player with id %d", results.InsertedID)
	return p, nil
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
	ctx := stream.Context()
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		log.Println(md)
	}
	for _, item := range items {
		_, span := otel.Tracer("PlayerServer").Start(ctx, "PlayerServer.SendItem")
		if err := stream.Send(&item); err != nil {
			return err
		}
		span.End()
	}
	// todo
	return nil
}

func (s *playerServer) newPlayer(ctx context.Context, name string, pType string) *proto.Player {
	_, span := otel.Tracer("PlayerServer").Start(ctx, "PlayerServer.newPlayer")
	defer span.End()
	return &proto.Player{Name: name, Health: 12}
}

func (s *playerServer) getPlayerByName(ctx context.Context, name string) *proto.Player {
	coll := s.getPlayerCollection(ctx)
	singleR := coll.FindOne(ctx, bson.D{{"name", name}})
	if singleR == nil || singleR.Err() != nil {
		return nil
	}
	var p proto.Player
	err := singleR.Decode(&p)
	if err != nil {
		log.Println(err)
		return nil
	}
	return &p
}

func (s *playerServer) getPlayerCollection(ctx context.Context) *mongo.Collection {
	return s.mongoClient.Database("rpcGame").Collection("players")
}
