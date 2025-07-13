package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/structpb"

	pk "github.com/spoungeai/spounge-proto/gen/go/polykey/v1"
	cmn "github.com/spoungeai/spounge-proto/gen/go/common/v1"
)

type mockService struct{}

func NewMockService() Service {
	return &mockService{}
}

func (s *mockService) ExecuteTool(ctx context.Context, req *pk.ExecuteToolRequest) (*pk.ExecuteToolResponse, error) {
	output, err := structpb.NewStruct(map[string]interface{}{
		"tool_executed": req.ToolName,
		"user_id":       req.UserId,
		"params_echo":   req.Parameters.AsMap(),
	})
	if err != nil {
		return nil, err
	}

	return &pk.ExecuteToolResponse{
		Status: &cmn.Status{
			Code:    0,
			Message: fmt.Sprintf("Tool '%s' executed successfully", req.ToolName),
		},
		Output: &pk.ExecuteToolResponse_StructOutput{
			StructOutput: output,
		},
		Metadata: &cmn.Metadata{
			Fields: map[string]string{
				"env": "mock",
			},
		},
	}, nil
}

func (s *mockService) ExecuteToolStream(req *pk.ExecuteToolStreamRequest, stream pk.PolykeyService_ExecuteToolStreamServer) error {
	for i := 1; i <= 3; i++ {
		resp := &pk.ExecuteToolStreamResponse{
			Status: &cmn.Status{Code: 0, Message: "ok"},
			Output: &pk.ExecuteToolStreamResponse_StringOutput{
				StringOutput: fmt.Sprintf("stream chunk %d for tool %s", i, req.ToolName),
			},
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}
