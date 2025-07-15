package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jpoz/llmite"
	"github.com/jpoz/llmite/anthropic"
)

func main() {
	// Create a new Anthropic client
	client := anthropic.New(
		anthropic.WithModel("claude-sonnet-4-20250514"),
		anthropic.WithMaxTokens(4000),
	)

	// Create a simple message
	messages := []llmite.Message{
		llmite.NewTextMessage(llmite.RoleSystem, "You are a helpful assistant."),
		llmite.NewTextMessage(llmite.RoleUser, "Write a detailed poem about programming in Go."),
	}

	// Create a streaming function that handles each chunk of the response
	streamFunc := func(response *llmite.Response, err error) bool {
		if err != nil {
			log.Printf("Stream error: %v", err)
			return false // Stop streaming on error
		}

		fmt.Print("\033[2J\033[H")

		fmt.Printf("Response ID: %s\n", response.ID)
		fmt.Printf("Provider: %s\n", response.Provider)
		fmt.Printf("Message Role: %s\n", response.Message.Role)
		fmt.Println("---")

		for _, part := range response.Message.Parts {
			switch p := part.(type) {
			case llmite.TextPart:
				fmt.Printf("%s", p.Text)
			case llmite.ToolCallPart:
				fmt.Printf("Tool Call: %s - %s\n", p.Name, p.Input)
			default:
				fmt.Printf("Unknown part type: %+v\n", p)
			}
		}

		return true // Continue streaming
	}

	// Generate a streaming response
	ctx := context.Background()
	finalResponse, err := client.GenerateStream(ctx, messages, streamFunc)
	if err != nil {
		log.Fatal(err)
	}

	// Print final response info
	fmt.Printf("\n\n=== FINAL RESPONSE ===\n")
	fmt.Printf("Response ID: %s\n", finalResponse.ID)
	fmt.Printf("Provider: %s\n", finalResponse.Provider)
	fmt.Printf("Total parts: %d\n", len(finalResponse.Message.Parts))
}

