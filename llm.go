package llmite

import (
	"context"
	"encoding/json"
)

type StreamFunc func(*Response, error) bool

type LLM interface {
	Generate(ctx context.Context, messages []Message) (*Response, error)
	GenerateStream(ctx context.Context, messages []Message, fn StreamFunc) (*Response, error)
}

type Part interface {
	isPart()
}

type Response struct {
	ID      string  `json:"id"`
	Message Message `json:"message"`

	Provider string
	Raw      any
}

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role  Role   `json:"role"`
	Parts []Part `json:"parts"`
}

func NewTextMessage(role Role, text string) Message {
	return Message{
		Role:  role,
		Parts: []Part{TextPart{Text: text}},
	}
}

type TextPart struct {
	Text string `json:"text"`
}

func (TextPart) isPart() {}

type ToolCallPart struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"arguments"`
}

func (ToolCallPart) isPart() {}

type ToolResultPart struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Result     any    `json:"result"`
	Error      error  `json:"error,omitempty"`
}

func (ToolResultPart) isPart() {}

// Tool defines the interface that all tools must implement
type Tool interface {
	// Name of the tool.
	//
	// This is how the tool will be called by the model and in `tool_use` blocks.
	Name() string

	// Description of what this tool does.
	//
	// Tool descriptions should be as detailed as possible. The more information that
	// the model has about what the tool is and how to use it, the better it will
	// perform. You can use natural language descriptions to reinforce important
	// aspects of the tool input JSON schema.
	Description() string

	// This defines the shape of the `input` that your tool accepts and that the model
	// will produce.
	Schema() map[string]interface{}
}

// ToolResult represents the result of tool execution
type ToolResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Error   error  `json:"error,omitempty"`
}
