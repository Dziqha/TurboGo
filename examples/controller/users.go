package controller

import (
	"encoding/json"
	"fmt"
	"github.com/Dziqha/TurboGo/core"
)

// --- Input struct ---
type CreateUserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// --- HTTP Handler ---
func CreateUserHandler(c *core.Context) {
	var input CreateUserInput

	if err := c.BindJSON(&input); err != nil {
		c.JSON(400, map[string]string{"error": "Invalid JSON"})
		return
	}

	// 🔁 Marshal payload ke []byte
	payload, err := json.Marshal(input)
	if err != nil {
		c.JSON(500, map[string]string{"error": "Failed to encode"})
		return
	}

	if err := c.MustQueue().EnqueueAll("user:welcome-email", payload); err != nil {
		c.JSON(500, map[string]string{"error": "Queue error"})
		return
	}
	if err := c.MustPubsub().PublishAll("user.created", payload); err != nil {
		c.JSON(500, map[string]string{"error": "Pubsub error"})
		return
	}
	c.JSON(201, map[string]string{"message": "User created"})
}

// --- Queue Worker ---
func SendWelcomeEmailWorker(data []byte) error {
	var input CreateUserInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}
	fmt.Printf("👷 Queue Worker: %s <%s>\n", input.Name, input.Email)
	return nil
}

func OnUserCreated(data []byte) error {
	var input CreateUserInput
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("invalid pubsub data: %w", err)
	}
	fmt.Printf("📣 PubSub Event: %s <%s>\n", input.Name, input.Email)
	return nil
}

func Quehandler(ctx *core.EngineContext) {
	if ctx.Queue == nil {
		panic("Queue engine not initialized")
	}
	ctx.Queue.RegisterWorkerAll("user:welcome-email", SendWelcomeEmailWorker)
}

func PubsubHandler(ps *core.EngineContext) {
	ch := ps.Pubsub.SubscribeAll("user.created")
	for msg := range ch {
		if err := OnUserCreated(msg); err != nil {
			fmt.Println("❌ pubsub handler error:", err)
		}
	}
}
