package llmite

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
