package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	cmd := exec.Command("protoc",
		"--proto_path=proto/llm",
		"--go_out=internal/adapters/llm",
		"--go_opt=paths=source_relative",
		"--go-grpc_out=internal/adapters/llm",
		"--go-grpc_opt=paths=source_relative",
		"proto/llm/llm.proto",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("protoc failed: %v", err)
	}
}
