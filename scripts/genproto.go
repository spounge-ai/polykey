package main

import (
    "log"
    "os"
    "os/exec"
)

func main() {
	// Can convert to slices later and then loop through them
    cmd := exec.Command("protoc",
        "--proto_path=internal/adapters/llm/providers/google",
        "--go_out=proto/google/gemini",
        "--go_opt=paths=source_relative",
        "--go-grpc_out=proto/google/gemini",
        "--go-grpc_opt=paths=source_relative",
        "internal/adapters/llm/providers/google/gemini.proto",
    )

    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        log.Fatalf("protoc failed: %v", err)
    }
}
