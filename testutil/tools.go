package testutil

import (
	"context"
	"encoding/json"

	"github.com/invopop/jsonschema"

	"github.com/jpoz/llmite"
)

type BoopTool struct{}

func NewBoopTool() llmite.Tool {
	return &BoopTool{}
}

func (t *BoopTool) Name() string {
	return "boop"
}

func (t *BoopTool) Description() string {
	return `This tool is used to translate boops for user. It takes a single parameter, "boops", which is a string containing the boops. The tool will return the string with the boops translated.`
}

type BoopToolParams struct {
	BoopString string `json:"boops" jsonschema:"title=Boop String,description=The string containing the boops"`
}

func (t *BoopTool) Schema() *jsonschema.Schema {
	return llmite.GenerateSchema[BoopToolParams]()
}

func (t *BoopTool) Execute(ctx context.Context, args []byte) *llmite.ToolResult {
	var params BoopToolParams
	err := json.Unmarshal(args, &params)
	if err != nil {
		return &llmite.ToolResult{
			ID:    "boop",
			Error: err,
		}
	}

	return &llmite.ToolResult{
		ID:      "boop",
		Content: `beep boop beep boop` + params.BoopString,
	}
}

