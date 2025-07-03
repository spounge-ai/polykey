package service_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/SpoungeAI/polykey-service/internal/service"
    pb "github.com/SpoungeAI/polykey-service/pkg/polykey/pb"
)

// TestCreateBot verifies the CreateBot method of the mock service.
func TestCreateBot(t *testing.T) {
    svc := service.NewMockService()
    ctx := context.Background()

    req := &pb.CreateBotRequest{
        Name:          "Test Bot",
        SystemPrompt:  "This is a test prompt.",
        ModelProvider: "openai",
    }

    bot, err := svc.CreateBot(ctx, req)

    require.NoError(t, err)
    require.NotNil(t, bot)

    assert.Equal(t, "mockbot123", bot.Id)
    assert.Equal(t, req.Name, bot.Name)
    assert.Equal(t, req.SystemPrompt, bot.SystemPrompt)
    assert.Equal(t, req.ModelProvider, bot.ModelProvider)
    assert.False(t, bot.ApiKeyIsSet)
    assert.NotNil(t, bot.CreatedAt)
    assert.NotNil(t, bot.UpdatedAt)
}

// TestGetBot verifies the GetBot method can retrieve a previously created bot.
func TestGetBot(t *testing.T) {
    svc := service.NewMockService()
    ctx := context.Background()

    createReq := &pb.CreateBotRequest{
        Name:          "Another Bot",
        SystemPrompt:  "Another test prompt.",
        ModelProvider: "groq",
    }
    createdBot, err := svc.CreateBot(ctx, createReq)
    require.NoError(t, err)
    require.NotNil(t, createdBot)

    retrievedBot, err := svc.GetBot(ctx, createdBot.Id)

    require.NoError(t, err)
    require.NotNil(t, retrievedBot)

    assert.Equal(t, createdBot.Id, retrievedBot.Id)
    assert.Equal(t, createdBot.Name, retrievedBot.Name)
}

// TestGetBot_NotFound verifies GetBot returns an error for a non-existent bot ID.
func TestGetBot_NotFound(t *testing.T) {
    svc := service.NewMockService()
    ctx := context.Background()

    nonExistentBotID := "nonexistent-id"
    bot, err := svc.GetBot(ctx, nonExistentBotID)

    require.Error(t, err)
    assert.Equal(t, service.ErrBotNotFound, err)
    assert.Nil(t, bot)
}

// Add tests for other methods in your Service interface as they are implemented.
// func TestUpdateBot(t *testing.T) { ... }
// func TestDeleteBot(t *testing.T) { ... }
// etc.