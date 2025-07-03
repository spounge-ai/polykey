package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pb "github.com/SpoungeAI/polykey-service/pkg/polykey/pb"
)

const (
    address = "localhost:50051" // Server address
)

func main() {
    log.Println("Starting dev_client...")

    conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    c := pb.NewPolyKeyClient(conn)

    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    createReq := &pb.CreateBotRequest{
        Name:          "Client Bot",
        SystemPrompt:  "Created from dev_client.",
        ModelProvider: "test",
    }
    log.Printf("Attempting to Create Bot: %v", createReq.Name)
    createdBot, err := c.CreateBot(ctx, createReq)
    if err != nil {
        log.Fatalf("could not create bot: %v", err)
    }
    log.Printf("Bot Created Successfully: %v (ID: %s)", createdBot.Name, createdBot.Id)

    getReq := &pb.GetBotRequest{BotId: createdBot.Id}
    log.Printf("Attempting to Get Bot with ID: %s", getReq.BotId)
    retrievedBot, err := c.GetBot(ctx, getReq)
    if err != nil {
        log.Fatalf("could not get bot: %v", err)
    }
    log.Printf("Bot Retrieved Successfully: %v (ID: %s)", retrievedBot.Name, retrievedBot.Id)

    log.Println("dev_client finished.")
}