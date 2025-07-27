package llms

import (
	"github.com/invopop/jsonschema"
)

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
	Schema() *jsonschema.Schema
}

// ToolResult represents the result of tool execution
type ToolResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Error   error  `json:"error,omitempty"`
}

// GenerateSchema generates a JSON schema for the given type T.
// This is a convenience wrapper around github.com/invopop/jsonschema that sets some
// reasonable defaults for LLM tools. It is recommended to use this function to
// generate the schema for your tool input types, but you can also construct the
// schema manually if you need more control.
func GenerateSchema[T any]() *jsonschema.Schema {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T

	return reflector.Reflect(v)
}
