package openai

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/jpoz/llmite"
	"github.com/jpoz/llmite/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIClientIntegration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	client := New(WithModel("gpt-4o-mini"))

	t.Run("simple generation", func(t *testing.T) {
		messages := []llmite.Message{
			{
				Role: llmite.RoleUser,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "What is the capital of France?"},
				},
			},
		}

		response, err := client.Generate(context.Background(), messages)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ID)
		assert.Equal(t, ProviderOpenAI, response.Provider)
		assert.Equal(t, llmite.RoleAssistant, response.Message.Role)
		assert.NotEmpty(t, response.Message.Parts)

		// Check that we got a text response
		hasText := false
		for _, part := range response.Message.Parts {
			if textPart, ok := part.(llmite.TextPart); ok {
				hasText = true
				assert.Contains(t, textPart.Text, "Paris")
			}
		}
		assert.True(t, hasText, "Expected text response")
	})

	t.Run("system message", func(t *testing.T) {
		messages := []llmite.Message{
			{
				Role: llmite.RoleSystem,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "You are a helpful assistant that always responds in exactly 5 words."},
				},
			},
			{
				Role: llmite.RoleUser,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "What is the capital of France?"},
				},
			},
		}

		response, err := client.Generate(context.Background(), messages)
		require.NoError(t, err)
		assert.NotNil(t, response)

		// Check that we got a short response
		hasText := false
		for _, part := range response.Message.Parts {
			if textPart, ok := part.(llmite.TextPart); ok {
				hasText = true
				assert.NotEmpty(t, textPart.Text)
			}
		}
		assert.True(t, hasText, "Expected text response")
	})

	t.Run("streaming", func(t *testing.T) {
		messages := []llmite.Message{
			{
				Role: llmite.RoleUser,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "Count from 1 to 5"},
				},
			},
		}

		var streamedResponses []*llmite.Response
		response, err := client.GenerateStream(context.Background(), messages, func(resp *llmite.Response, err error) bool {
			if err != nil {
				return false
			}
			streamedResponses = append(streamedResponses, resp)
			return true
		})

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.ID)

		// Check that we received streamed responses
		assert.NotEmpty(t, streamedResponses, "Expected streamed responses")

		// Check that the final response contains the content
		hasText := false
		for _, part := range response.Message.Parts {
			if _, ok := part.(llmite.TextPart); ok {
				hasText = true
			}
		}
		assert.True(t, hasText, "Expected text in final response")
	})

	t.Run("tool calling", func(t *testing.T) {
		tools := []llmite.Tool{
			testutil.WeatherTool{},
		}

		clientWithTools := New(
			WithModel("gpt-4o-mini"),
			WithTools(tools),
		)

		messages := []llmite.Message{
			{
				Role: llmite.RoleUser,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "What's the weather like in San Francisco?"},
				},
			},
		}

		response, err := clientWithTools.Generate(context.Background(), messages)
		require.NoError(t, err)
		assert.NotNil(t, response)

		// Check if we got a tool call
		hasToolCall := false
		for _, part := range response.Message.Parts {
			if toolCallPart, ok := part.(llmite.ToolCallPart); ok {
				hasToolCall = true
				assert.Equal(t, "get_weather", toolCallPart.Name)
				assert.NotEmpty(t, toolCallPart.ID)
				assert.NotEmpty(t, toolCallPart.Input)
			}
		}

		// Note: The model may choose to respond with text instead of a tool call
		// depending on the prompt and model behavior
		log.Printf("Got tool call: %v", hasToolCall)
	})

	t.Run("conversation with tool result", func(t *testing.T) {
		tools := []llmite.Tool{
			testutil.WeatherTool{},
		}

		clientWithTools := New(
			WithModel("gpt-4o-mini"),
			WithTools(tools),
		)

		messages := []llmite.Message{
			{
				Role: llmite.RoleUser,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "What's the weather like in San Francisco?"},
				},
			},
			{
				Role: llmite.RoleAssistant,
				Parts: []llmite.Part{
					llmite.ToolCallPart{
						ID:    "call_123",
						Name:  "get_weather",
						Input: []byte(`{"location": "San Francisco"}`),
					},
				},
			},
			{
				Role: llmite.RoleAssistant,
				Parts: []llmite.Part{
					llmite.ToolResultPart{
						ToolCallID: "call_123",
						Name:       "get_weather",
						Result:     "The weather in San Francisco is sunny, 72°F",
					},
				},
			},
			{
				Role: llmite.RoleUser,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "Is that good weather for a picnic?"},
				},
			},
		}

		response, err := clientWithTools.Generate(context.Background(), messages)
		require.NoError(t, err)
		assert.NotNil(t, response)

		// Should get a text response about the weather being good for a picnic
		hasText := false
		for _, part := range response.Message.Parts {
			if textPart, ok := part.(llmite.TextPart); ok {
				hasText = true
				assert.NotEmpty(t, textPart.Text)
			}
		}
		assert.True(t, hasText, "Expected text response")
	})
}

func TestOpenAIClientConfiguration(t *testing.T) {
	t.Run("with custom model", func(t *testing.T) {
		client := New(WithModel("gpt-3.5-turbo"))
		oaiClient := client.(*Client)
		assert.Equal(t, "gpt-3.5-turbo", oaiClient.Model)
	})

	t.Run("with custom max tokens", func(t *testing.T) {
		client := New(WithMaxTokens(2048))
		oaiClient := client.(*Client)
		assert.Equal(t, int64(2048), oaiClient.MaxTokens)
	})

	t.Run("with temperature", func(t *testing.T) {
		client := New(WithTemperature(0.7))
		oaiClient := client.(*Client)
		assert.NotNil(t, oaiClient.Temperature)
		assert.Equal(t, 0.7, *oaiClient.Temperature)
	})

	t.Run("with top_p", func(t *testing.T) {
		client := New(WithTopP(0.9))
		oaiClient := client.(*Client)
		assert.NotNil(t, oaiClient.TopP)
		assert.Equal(t, 0.9, *oaiClient.TopP)
	})

	t.Run("with tools", func(t *testing.T) {
		tools := []llmite.Tool{testutil.WeatherTool{}}
		client := New(WithTools(tools))
		oaiClient := client.(*Client)
		assert.Equal(t, tools, oaiClient.Tools)
	})
}