package server

import (
	"context"
	"log/slog"

	"github.com/spounge-ai/polykey-service/internal/service"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
)

// Server implements the PolykeyServiceServer interface
type Server struct {
	pk.UnimplementedPolykeyServiceServer
	service service.Service
	logger  *slog.Logger
}

// NewServer creates a new server instance
func NewServer(service service.Service) *Server {
	return &Server{
		service: service,
		logger:  slog.Default(),
	}
}

// ExecuteTool implements the ExecuteTool RPC method
func (s *Server) ExecuteTool(ctx context.Context, req *pk.ExecuteToolRequest) (*pk.ExecuteToolResponse, error) {
	s.logger.Info("ExecuteTool called", 
		"tool_name", req.ToolName,
		"has_parameters", req.Parameters != nil,
		"has_secret_id", req.SecretId != nil,
		"has_metadata", req.Metadata != nil,
	)

	// Call the underlying service
	resp, err := s.service.ExecuteTool(ctx, req.ToolName, req.Parameters, req.SecretId, req.Metadata)
	if err != nil {
		s.logger.Error("Service ExecuteTool failed", "error", err)
		return nil, err
	}

	return resp, nil
}