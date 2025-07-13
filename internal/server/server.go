package server

import (
	"context"

	pk "github.com/spoungeai/spounge-proto/gen/go/polykey/v1"
	"github.com/SpoungeAI/polykey-service/internal/service"
)

type Server struct {
	pk.UnimplementedPolykeyServiceServer
	svc service.Service
}

func NewServer(svc service.Service) *Server {
	return &Server{svc: svc}
}

func (s *Server) ExecuteTool(ctx context.Context, req *pk.ExecuteToolRequest) (*pk.ExecuteToolResponse, error) {
	return s.svc.ExecuteTool(ctx, req)
}

func (s *Server) ExecuteToolStream(req *pk.ExecuteToolStreamRequest, stream pk.PolykeyService_ExecuteToolStreamServer) error {
	// stub
	return nil
}
