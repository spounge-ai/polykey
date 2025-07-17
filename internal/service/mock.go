// File: internal/service/mock.go
package service

import (
	"context"
	"fmt"
	"time"

	cmn "github.com/spounge-ai/spounge-proto/gen/go/common/v2"
	pk "github.com/spounge-ai/spounge-proto/gen/go/polykey/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

type MockService struct{}

// NewMockService creates a new mock service instance
func NewMockService() *MockService {
	return &MockService{}
}

// ExecuteTool implements the Service interface
func (m *MockService) ExecuteTool(ctx context.Context, toolName string, parameters *structpb.Struct, secretId *string, metadata *cmn.Metadata) (*pk.ExecuteToolResponse, error) {
	// Create a successful response
	response := &pk.ExecuteToolResponse{
		Status: &cmn.Status{
			Code:    200,
			Message: "Tool executed successfully",
		},
	}

	// Based on the tool name, return different types of output
	switch toolName {
	case "example_tool":
		response.Output = &pk.ExecuteToolResponse_StringOutput{
			StringOutput: fmt.Sprintf("Mock execution of %s at %s", toolName, time.Now().Format(time.RFC3339)),
		}
	case "struct_tool":
		structOutput, err := structpb.NewStruct(map[string]interface{}{
			"result":    "success",
			"timestamp": time.Now().Unix(),
			"data":      map[string]interface{}{
				"processed": true,
				"count":     42,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create struct output: %w", err)
		}
		response.Output = &pk.ExecuteToolResponse_StructOutput{
			StructOutput: structOutput,
		}
	case "file_tool":
		response.Output = &pk.ExecuteToolResponse_FileOutput{
			FileOutput: &cmn.File{
				FileName: "example.txt",
				MimeType: "text/plain",
				Content:  []byte("This is mock file content"),
			},
		}
	default:
		response.Output = &pk.ExecuteToolResponse_StringOutput{
			StringOutput: fmt.Sprintf("Unknown tool: %s", toolName),
		}
	}

	return response, nil
}