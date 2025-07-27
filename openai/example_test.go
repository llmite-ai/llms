package openai_test

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/llmite-ai/llms"
	"github.com/llmite-ai/llms/openai"
)

func ExampleClient_Generate() {
	// Create OpenAI client
	client := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithMaxTokens(100),
		openai.WithTemperature(0.7),
	)

	// Create messages
	messages := []llms.Message{
		{
			Role: llms.RoleSystem,
			Parts: []llms.Part{
				llms.TextPart{Text: "You are a helpful assistant."},
			},
		},
		{
			Role: llms.RoleUser,
			Parts: []llms.Part{
				llms.TextPart{Text: "What is the capital of France?"},
			},
		},
	}

	// Generate response
	response, err := client.Generate(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}

	// Print the response
	for _, part := range response.Message.Parts {
		if textPart, ok := part.(llms.TextPart); ok {
			fmt.Println(textPart.Text)
		}
	}
}

func ExampleClient_GenerateStream() {
	// Create OpenAI client
	client := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithMaxTokens(100),
	)

	// Create messages
	messages := []llms.Message{
		{
			Role: llms.RoleUser,
			Parts: []llms.Part{
				llms.TextPart{Text: "Count from 1 to 5"},
			},
		},
	}

	// Generate streaming response
	_, err := client.GenerateStream(context.Background(), messages, func(response *llms.Response, err error) bool {
		if err != nil {
			log.Printf("Streaming error: %v", err)
			return false
		}

		// Process each part in the response
		for _, part := range response.Message.Parts {
			if textPart, ok := part.(llms.TextPart); ok {
				fmt.Print(textPart.Text)
			}
		}

		return true // Continue streaming
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleClient_WithLogging() {
	// Create OpenAI client with HTTP logging
	client := openai.New(
		openai.WithModel("gpt-4o-mini"),
		openai.WithHttpLogging(),
	)

	// Set the API key
	os.Setenv("OPENAI_API_KEY", "your-api-key-here")

	// Create messages
	messages := []llms.Message{
		{
			Role: llms.RoleUser,
			Parts: []llms.Part{
				llms.TextPart{Text: "Hello, world!"},
			},
		},
	}

	// Generate response with logging
	response, err := client.Generate(context.Background(), messages)
	if err != nil {
		log.Fatal(err)
	}

	// Print the response
	for _, part := range response.Message.Parts {
		if textPart, ok := part.(llms.TextPart); ok {
			fmt.Println(textPart.Text)
		}
	}
}