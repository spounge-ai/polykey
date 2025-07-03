package service

import (
	"context"
	"errors"

	pb "github.com/SpoungeAI/polykey-service/pkg/polykey/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrBotNotFound = errors.New("bot not found")
)

type Service interface {
	CreateBot(ctx context.Context, req *pb.CreateBotRequest) (*pb.Bot, error)
	GetBot(ctx context.Context, botID string) (*pb.Bot, error)
	
}

type mockService struct {
	bots map[string]*pb.Bot
}

func NewMockService() Service {
	return &mockService{
		bots: make(map[string]*pb.Bot),
	}
}

func (m *mockService) CreateBot(ctx context.Context, req *pb.CreateBotRequest) (*pb.Bot, error) {
	bot := &pb.Bot{
		Id:            "mockbot123",
		Name:          req.Name,
		SystemPrompt:  req.SystemPrompt,
		ModelProvider: req.ModelProvider,
		ApiKeyIsSet:   false,
		CreatedAt:     timestampNow(),
		UpdatedAt:     timestampNow(),
	}
	m.bots[bot.Id] = bot
	return bot, nil
}

func (m *mockService) GetBot(ctx context.Context, botID string) (*pb.Bot, error) {
	bot, ok := m.bots[botID]
	if !ok {
		return nil, ErrBotNotFound
	}
	return bot, nil
}

func timestampNow() *timestamppb.Timestamp {
	return timestamppb.Now()
}

