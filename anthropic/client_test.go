package anthropic

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"

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
					assert.Equal(t, anthropic.MessageParamRoleUser, msg.Role)
				case llmite.RoleAssistant:
					assert.Equal(t, anthropic.MessageParamRoleAssistant, msg.Role)
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
	assert.Equal(t, anthropic.MessageParamRoleUser, anthMessages[0].Role)
}

// Mock tool for testing
type mockTool struct {
	name        string
	description string
	schema      *jsonschema.Schema
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return m.description
}

func (m *mockTool) Schema() *jsonschema.Schema {
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
					schema: &jsonschema.Schema{
						Type: "object",
						Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
							props := orderedmap.New[string, *jsonschema.Schema]()
							props.Set("input", &jsonschema.Schema{Type: "string"})
							return props
						}(),
						Required: []string{"input"},
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
					schema: &jsonschema.Schema{
						Type: "object",
						Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
							props := orderedmap.New[string, *jsonschema.Schema]()
							props.Set("expression", &jsonschema.Schema{Type: "string"})
							return props
						}(),
						Required: []string{"expression"},
					},
				},
				&mockTool{
					name:        "weather",
					description: "Get weather information",
					schema: &jsonschema.Schema{
						Type: "object",
						Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
							props := orderedmap.New[string, *jsonschema.Schema]()
							props.Set("location", &jsonschema.Schema{Type: "string"})
							props.Set("unit", &jsonschema.Schema{Type: "string", Enum: []interface{}{"celsius", "fahrenheit"}})
							return props
						}(),
						Required: []string{"location"},
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
				assert.NotNil(t, tool.OfTool)
				assert.Equal(t, tt.tools[i].Name(), tool.OfTool.Name)
				if tool.OfTool.Description.Valid() {
					assert.Equal(t, tt.tools[i].Description(), tool.OfTool.Description.Value)
				}

				// Verify schema conversion
				originalSchema := tt.tools[i].Schema()
				assert.Equal(t, originalSchema.Properties, tool.OfTool.InputSchema.Properties)
				assert.Equal(t, originalSchema.Required, tool.OfTool.InputSchema.Required)
			}
		})
	}
}

func TestConvertTools_SchemaValidation(t *testing.T) {
	tool := &mockTool{
		name:        "complex_tool",
		description: "A tool with complex schema",
		schema: &jsonschema.Schema{
			Type: "object",
			Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
				props := orderedmap.New[string, *jsonschema.Schema]()
				props.Set("string_field", &jsonschema.Schema{Type: "string"})
				props.Set("number_field", &jsonschema.Schema{Type: "number"})
				props.Set("boolean_field", &jsonschema.Schema{Type: "boolean"})
				props.Set("array_field", &jsonschema.Schema{
					Type:  "array",
					Items: &jsonschema.Schema{Type: "string"},
				})
				nestedProps := orderedmap.New[string, *jsonschema.Schema]()
				nestedProps.Set("nested", &jsonschema.Schema{Type: "string"})
				props.Set("object_field", &jsonschema.Schema{
					Type:       "object",
					Properties: nestedProps,
				})
				return props
			}(),
			Required: []string{"string_field", "number_field"},
		},
	}

	result, err := convertTools([]llmite.Tool{tool})
	require.NoError(t, err)
	require.Len(t, result, 1)

	anthTool := result[0].OfTool
	assert.Equal(t, "complex_tool", anthTool.Name)
	if anthTool.Description.Valid() {
		assert.Equal(t, "A tool with complex schema", anthTool.Description.Value)
	}

	schema := anthTool.InputSchema
	assert.Equal(t, tool.schema.Properties, schema.Properties)
	assert.Equal(t, tool.schema.Required, schema.Required)
}
