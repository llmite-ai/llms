package openai

import (
	"testing"

	"github.com/llmite-ai/llms"
	"github.com/llmite-ai/llms/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertMessages(t *testing.T) {
	t.Run("system message", func(t *testing.T) {
		messages := []llmite.Message{
			{
				Role: llmite.RoleSystem,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "You are a helpful assistant."},
				},
			},
		}

		result, err := convertMessages(messages)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		// The result should be a system message created by openai.SystemMessage()
	})

	t.Run("user message", func(t *testing.T) {
		messages := []llmite.Message{
			{
				Role: llmite.RoleUser,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "Hello, world!"},
				},
			},
		}

		result, err := convertMessages(messages)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		// The result should be a user message created by openai.UserMessage()
	})

	t.Run("assistant message with text", func(t *testing.T) {
		messages := []llmite.Message{
			{
				Role: llmite.RoleAssistant,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "Hello!"},
				},
			},
		}

		result, err := convertMessages(messages)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		// The result should be an assistant message created by openai.AssistantMessage()
	})

	t.Run("assistant message with tool call", func(t *testing.T) {
		messages := []llmite.Message{
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
		}

		result, err := convertMessages(messages)
		require.NoError(t, err)
		// Tool calls are not yet implemented, so we expect 0 messages
		assert.Len(t, result, 0)
	})

	t.Run("tool result message", func(t *testing.T) {
		messages := []llmite.Message{
			{
				Role: llmite.RoleAssistant,
				Parts: []llmite.Part{
					llmite.ToolResultPart{
						ToolCallID: "call_123",
						Name:       "get_weather",
						Result:     "Sunny, 72°F",
					},
				},
			},
		}

		result, err := convertMessages(messages)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		// The result should be a tool message created by openai.ToolMessage()
	})

	t.Run("mixed message types", func(t *testing.T) {
		messages := []llmite.Message{
			{
				Role: llmite.RoleSystem,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "You are helpful."},
				},
			},
			{
				Role: llmite.RoleUser,
				Parts: []llmite.Part{
					llmite.TextPart{Text: "What's the weather?"},
				},
			},
			{
				Role: llmite.RoleAssistant,
				Parts: []llmite.Part{
					llmite.ToolCallPart{
						ID:    "call_123",
						Name:  "get_weather",
						Input: []byte(`{"location": "default"}`),
					},
				},
			},
		}

		result, err := convertMessages(messages)
		require.NoError(t, err)
		// Tool calls are not yet implemented, so we expect 2 messages (system and user)
		assert.Len(t, result, 2)
	})
}

func TestConvertTools(t *testing.T) {
	t.Run("no tools", func(t *testing.T) {
		result, err := convertTools(nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty tools", func(t *testing.T) {
		result, err := convertTools([]llmite.Tool{})
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("single tool", func(t *testing.T) {
		tools := []llmite.Tool{
			testutil.WeatherTool{},
		}

		result, err := convertTools(tools)
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "get_weather", result[0].Function.Name)
		// Check if description is set (simplified check)
		assert.NotNil(t, result[0].Function.Description)
		assert.NotNil(t, result[0].Function.Parameters)
	})

	t.Run("multiple tools", func(t *testing.T) {
		tools := []llmite.Tool{
			testutil.WeatherTool{},
			testutil.CalculatorTool{},
		}

		result, err := convertTools(tools)
		require.NoError(t, err)
		assert.Len(t, result, 2)
		
		// Check that both tools are present
		toolNames := make(map[string]bool)
		for _, tool := range result {
			toolNames[tool.Function.Name] = true
		}
		assert.True(t, toolNames["get_weather"])
		assert.True(t, toolNames["calculate"])
	})
}

func TestClientDefaults(t *testing.T) {
	client := New()
	oaiClient := client.(*Client)
	
	assert.Equal(t, "gpt-4o", oaiClient.Model)
	assert.Equal(t, int64(1024), oaiClient.MaxTokens)
	assert.Nil(t, oaiClient.Temperature)
	assert.Nil(t, oaiClient.TopP)
	assert.Nil(t, oaiClient.Tools)
	assert.NotNil(t, oaiClient.client)
}