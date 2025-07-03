package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/SpoungeAI/polykey-service/pkg/polykey/pb"
	"github.com/SpoungeAI/polykey-service/internal/service"
)

type server struct {
	pb.UnimplementedPolyKeyServer
	service service.Service
}

func NewServer(svc service.Service) *server {
	return &server{
		service: svc,
	}
}

func (s *server) CreateBot(ctx context.Context, req *pb.CreateBotRequest) (*pb.Bot, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	bot, err := s.service.CreateBot(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create bot: %v", err)
	}
	return bot, nil
}

func (s *server) GetBot(ctx context.Context, req *pb.GetBotRequest) (*pb.Bot, error) {
	bot, err := s.service.GetBot(ctx, req.BotId)
	if err != nil {
		if err == service.ErrBotNotFound {
			return nil, status.Error(codes.NotFound, "bot not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get bot: %v", err)
	}
	return bot, nil
}

// Stub other methods returning Unimplemented error for now
func (s *server) RouteChat(req *pb.RouteChatRequest, stream pb.PolyKey_RouteChatServer) error {
	return status.Error(codes.Unimplemented, "RouteChat not implemented")
}

func (s *server) UpdateBot(ctx context.Context, req *pb.UpdateBotRequest) (*pb.Bot, error) {
	return nil, status.Error(codes.Unimplemented, "UpdateBot not implemented")
}
func (s *server) DeleteBot(ctx context.Context, req *pb.DeleteBotRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, status.Error(codes.Unimplemented, "DeleteBot not implemented")
}


func (s *server) ListBots(req *pb.ListBotsRequest, stream pb.PolyKey_ListBotsServer) error {
	return status.Error(codes.Unimplemented, "ListBots not implemented")
}
func (s *server) SetBotAPIKey(ctx context.Context, req *pb.SetBotAPIKeyRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, status.Error(codes.Unimplemented, "SetBotAPIKey not implemented")
}

