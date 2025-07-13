package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pkv1 "github.com/spoungeai/spounge-proto/gen/go/polykey/v1"
    "google.golang.org/protobuf/types/known/structpb"
)

const (
    address = "localhost:50051"
)

func main() {
    log.Println("Starting dev_client...")

    // Connect to gRPC server
    conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pkv1.NewPolykeyServiceClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Prepare parameters as a protobuf Struct (empty here, but add your fields as needed)
    params, err := structpb.NewStruct(map[string]interface{}{
        "example_param": "value",
    })
    if err != nil {
        log.Fatalf("failed to create parameters struct: %v", err)
    }

    req := &pkv1.ExecuteToolRequest{
        ToolName:   "example_tool",
        Parameters: params,
        UserId:     "user-123",
        // WorkflowRunId and Metadata can be added if you have them
    }

    log.Printf("Calling ExecuteTool with tool_name: %s", req.ToolName)

    resp, err := client.ExecuteTool(ctx, req)
    if err != nil {
        log.Fatalf("ExecuteTool failed: %v", err)
    }

    log.Printf("ExecuteTool status: %v", resp.Status)

    // Handle the output oneof field
    switch output := resp.Output.(type) {
    case *pkv1.ExecuteToolResponse_StringOutput:
        log.Printf("String Output: %s", output.StringOutput)
    case *pkv1.ExecuteToolResponse_StructOutput:
        log.Printf("Struct Output: %v", output.StructOutput)
    case *pkv1.ExecuteToolResponse_FileOutput:
        log.Printf("File Output: %+v", output.FileOutput)
    default:
        log.Println("No output returned")
    }

    log.Println("dev_client finished.")
}
