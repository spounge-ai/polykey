package main

import (
    "context"
    "log"
    "os"
    "strings"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pk "github.com/spoungeai/spounge-proto/gen/go/polykey/v1"
    "google.golang.org/protobuf/types/known/structpb"
)

func main() {
    log.Println("Starting dev_client...")

    address := getServerAddress()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()

    client := pk.NewPolykeyServiceClient(conn)

    params, err := structpb.NewStruct(map[string]interface{}{
        "example_param": "value",
    })
    if err != nil {
        log.Fatalf("failed to create parameters struct: %v", err)
    }

    req := &pk.ExecuteToolRequest{
        ToolName:   "example_tool",
        Parameters: params,
        UserId:     "user-123",
    }

    log.Printf("Calling ExecuteTool with tool_name: %s", req.ToolName)

    resp, err := client.ExecuteTool(ctx, req)
    if err != nil {
        log.Fatalf("ExecuteTool failed: %v", err)
    }

    log.Printf("ExecuteTool status: %v", resp.Status)

    switch output := resp.Output.(type) {
    case *pk.ExecuteToolResponse_StringOutput:
        log.Printf("String Output: %s", output.StringOutput)
    case *pk.ExecuteToolResponse_StructOutput:
        log.Printf("Struct Output: %v", output.StructOutput)
    case *pk.ExecuteToolResponse_FileOutput:
        log.Printf("File Output: %+v", output.FileOutput)
    default:
        log.Println("No output returned")
    }

    log.Println("dev_client finished.")
}

func getServerAddress() string {
    if envAddr := os.Getenv("POLYKEY_SERVER_ADDR"); envAddr != "" {
        return envAddr
    }

    if isRunningInDocker() {
        return "polykey-server:50051" // When both client and server are in Docker Compose
    }

    if isDockerHostReachable() {
        return "host.docker.internal:50052" // Local client, server in Docker
    }

    return "localhost:50051" // Fallback to local
}

func isRunningInDocker() bool {
    if _, err := os.Stat("/.dockerenv"); err == nil {
        return true
    }

    data, err := os.ReadFile("/proc/1/cgroup")
    if err != nil {
        return false
    }

    return strings.Contains(string(data), "docker")
}

func isDockerHostReachable() bool {
    // Assume Docker host is reachable (e.g., via Docker Desktop)
    return true
}
