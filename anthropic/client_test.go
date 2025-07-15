package anthropic

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jpoz/llmite"
)

func TestConvertMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []llmite.Message
		wantErr  bool
	}{
		{
			name: "single text message",
			messages: []llmite.Message{
				{
					Role:  llmite.RoleUser,
					Parts: []llmite.Part{llmite.TextPart{Text: "Hello"}},
				},
			},
			wantErr: false,
		},
		{
			name: "system message",
			messages: []llmite.Message{
				{
					Role:  llmite.RoleSystem,
					Parts: []llmite.Part{llmite.TextPart{Text: "You are a helpful assistant"}},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple messages with different roles",
			messages: []llmite.Message{
				{
					Role:  llmite.RoleSystem,
					Parts: []llmite.Part{llmite.TextPart{Text: "You are a helpful assistant"}},
				},
				{
					Role:  llmite.RoleUser,
					Parts: []llmite.Part{llmite.TextPart{Text: "Hello"}},
				},
				{
					Role:  llmite.RoleAssistant,
					Parts: []llmite.Part{llmite.TextPart{Text: "Hi there!"}},
				},
			},
			wantErr: false,
		},
		{
			name: "tool call message",
			messages: []llmite.Message{
				{
					Role: llmite.RoleAssistant,
					Parts: []llmite.Part{
						llmite.ToolCallPart{
							ID:    "call_123",
							Name:  "test_tool",
							Input: []byte(`{"arg": "value"}`),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "tool result message",
			messages: []llmite.Message{
				{
					Role: llmite.RoleUser,
					Parts: []llmite.Part{
						llmite.ToolResultPart{
							ToolCallID: "call_123",
							Name:       "test_tool",
							Result:     "success",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "tool result with error",
			messages: []llmite.Message{
				{
					Role: llmite.RoleUser,
					Parts: []llmite.Part{
						llmite.ToolResultPart{
							ToolCallID: "call_123",
							Name:       "test_tool",
							Result:     "failed",
							Error:      fmt.Errorf("test error"),
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "mixed content message",
			messages: []llmite.Message{
				{
					Role: llmite.RoleUser,
					Parts: []llmite.Part{
						llmite.TextPart{Text: "Please use this tool:"},
						llmite.ToolCallPart{
							ID:    "call_456",
							Name:  "calculator",
							Input: []byte(`{"expression": "2+2"}`),
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			system, messages, err := convertMessages(tt.messages)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Check that system messages are properly extracted
			systemCount := 0
			for _, msg := range tt.messages {
				if msg.Role == llmite.RoleSystem {
					systemCount++
				}
			}

			if systemCount > 0 {
				assert.NotEmpty(t, system)
			}

			// Check that non-system messages are properly converted
			nonSystemCount := 0
			for _, msg := range tt.messages {
				if msg.Role != llmite.RoleSystem {
					nonSystemCount++
				}
			}

			assert.Len(t, messages, nonSystemCount)

			// Verify message structure for non-system messages
			nonSystemIndex := 0
			for _, originalMsg := range tt.messages {
				if originalMsg.Role == llmite.RoleSystem {
					continue
				}

				msg := messages[nonSystemIndex]
				assert.NotEmpty(t, msg.Role)
				assert.NotEmpty(t, msg.Content)

				// Check that role conversion is correct
				switch originalMsg.Role {
				case llmite.RoleUser:
					assert.Equal(t, "user", msg.Role)
				case llmite.RoleAssistant:
					assert.Equal(t, "assistant", msg.Role)
				}

				nonSystemIndex++
			}
		})
	}
}

func TestConvertMessages_SystemMessages(t *testing.T) {
	messages := []llmite.Message{
		{
			Role:  llmite.RoleSystem,
			Parts: []llmite.Part{llmite.TextPart{Text: "First system message"}},
		},
		{
			Role:  llmite.RoleSystem,
			Parts: []llmite.Part{llmite.TextPart{Text: "Second system message"}},
		},
		{
			Role:  llmite.RoleUser,
			Parts: []llmite.Part{llmite.TextPart{Text: "User message"}},
		},
	}

	system, anthMessages, err := convertMessages(messages)
	require.NoError(t, err)

	assert.Len(t, system, 2)
	assert.Equal(t, "First system message", system[0].Text)
	assert.Equal(t, "Second system message", system[1].Text)
	assert.Len(t, anthMessages, 1)
	assert.Equal(t, "user", anthMessages[0].Role)
}

// Mock tool for testing
type mockTool struct {
	name        string
	description string
	schema      map[string]interface{}
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Schema() map[string]interface{} {
	return m.schema
}

func (m *mockTool) Execute(ctx context.Context, args []byte) *llmite.ToolResult {
	return &llmite.ToolResult{
		ID:      "test_result",
		Content: "mock result",
	}
}

func TestConvertTools(t *testing.T) {
	tests := []struct {
		name    string
		tools   []llmite.Tool
		wantErr bool
	}{
		{
			name:    "empty tools",
			tools:   []llmite.Tool{},
			wantErr: false,
		},
		{
			name: "single tool",
			tools: []llmite.Tool{
				&mockTool{
					name:        "test_tool",
					description: "A test tool",
					schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"input": map[string]interface{}{
								"type": "string",
							},
						},
						"required": []string{"input"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple tools",
			tools: []llmite.Tool{
				&mockTool{
					name:        "calculator",
					description: "Calculate mathematical expressions",
					schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"expression": map[string]interface{}{
								"type": "string",
							},
						},
						"required": []string{"expression"},
					},
				},
				&mockTool{
					name:        "weather",
					description: "Get weather information",
					schema: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type": "string",
							},
							"unit": map[string]interface{}{
								"type": "string",
								"enum": []interface{}{"celsius", "fahrenheit"},
							},
						},
						"required": []string{"location"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertTools(tt.tools)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, len(tt.tools))

			for i, tool := range result {
				assert.NotNil(t, tool)
				assert.Equal(t, tt.tools[i].Name(), tool.Name)
				if tool.Description != nil {
					assert.Equal(t, tt.tools[i].Description(), *tool.Description)
				}

				// Verify schema conversion
				originalSchema := tt.tools[i].Schema()
				assert.Equal(t, originalSchema["properties"], tool.InputSchema.Properties)
				assert.Equal(t, originalSchema["required"], tool.InputSchema.Required)
			}
		})
	}
}

func TestConvertTools_SchemaValidation(t *testing.T) {
	tool := &mockTool{
		name:        "complex_tool",
		description: "A tool with complex schema",
		schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"string_field": map[string]interface{}{
					"type": "string",
				},
				"number_field": map[string]interface{}{
					"type": "number",
				},
				"boolean_field": map[string]interface{}{
					"type": "boolean",
				},
				"array_field": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"object_field": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"nested": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			"required": []string{"string_field", "number_field"},
		},
	}

	result, err := convertTools([]llmite.Tool{tool})
	require.NoError(t, err)
	require.Len(t, result, 1)

	anthTool := result[0]
	assert.Equal(t, "complex_tool", anthTool.Name)
	if anthTool.Description != nil {
		assert.Equal(t, "A tool with complex schema", *anthTool.Description)
	}

	schema := anthTool.InputSchema
	assert.Equal(t, tool.schema["properties"], schema.Properties)
	assert.Equal(t, tool.schema["required"], schema.Required)
}
