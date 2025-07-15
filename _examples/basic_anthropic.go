package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jpoz/llmite"
	"github.com/jpoz/llmite/anthropic"
)

func main() {
	// Create a new Anthropic client
	client := anthropic.New(
		anthropic.WithModel("claude-sonnet-4-20250514"),
		anthropic.WithMaxTokens(1000),
	)

	// Create a simple message
	messages := []llmite.Message{
		llmite.NewTextMessage(llmite.RoleSystem, "You are a helpful assistant."),
		llmite.NewTextMessage(llmite.RoleUser, "Hello! Can you tell me about Go programming?"),
	}

	// Generate a response
	ctx := context.Background()
	response, err := client.Generate(ctx, messages)
	if err != nil {
		log.Fatal(err)
	}

	// Print the response
	fmt.Printf("Response ID: %s\n", response.ID)
	fmt.Printf("Provider: %s\n", response.Provider)
	fmt.Printf("Message Role: %s\n", response.Message.Role)
	
	// Print all parts of the response
	for i, part := range response.Message.Parts {
		switch p := part.(type) {
		case llmite.TextPart:
			fmt.Printf("Part %d (Text): %s\n", i, p.Text)
		case llmite.ToolCallPart:
			fmt.Printf("Part %d (Tool Call): %s - %s\n", i, p.Name, p.Input)
		default:
			fmt.Printf("Part %d (Unknown): %+v\n", i, p)
		}
	}
}