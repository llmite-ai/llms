package testutil

import (
	"context"
	"encoding/json"

	"github.com/invopop/jsonschema"

	"github.com/llmite-ai/llms"
)

type BoopTool struct{}

func NewBoopTool() llms.Tool {
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
	return llms.GenerateSchema[BoopToolParams]()
}

func (t *BoopTool) Execute(ctx context.Context, args []byte) *llms.ToolResult {
	var params BoopToolParams
	err := json.Unmarshal(args, &params)
	if err != nil {
		return &llms.ToolResult{
			ID:    "boop",
			Error: err,
		}
	}

	return &llms.ToolResult{
		ID:      "boop",
		Content: `beep boop beep boop` + params.BoopString,
	}
}

// WeatherTool is a simple weather tool for testing
type WeatherTool struct{}

func (t WeatherTool) Name() string {
	return "get_weather"
}

func (t WeatherTool) Description() string {
	return "Get the current weather for a location"
}

type WeatherToolParams struct {
	Location string `json:"location" jsonschema:"title=Location,description=The location to get weather for"`
}

func (t WeatherTool) Schema() *jsonschema.Schema {
	return llms.GenerateSchema[WeatherToolParams]()
}

func (t WeatherTool) Execute(ctx context.Context, args []byte) *llms.ToolResult {
	var params WeatherToolParams
	err := json.Unmarshal(args, &params)
	if err != nil {
		return &llms.ToolResult{
			ID:    "weather",
			Error: err,
		}
	}

	return &llms.ToolResult{
		ID:      "weather",
		Content: `The weather in ` + params.Location + ` is sunny, 72°F`,
	}
}

// CalculatorTool is a simple calculator tool for testing
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculate"
}

func (t CalculatorTool) Description() string {
	return "Perform basic arithmetic calculations"
}

type CalculatorToolParams struct {
	Expression string `json:"expression" jsonschema:"title=Expression,description=The mathematical expression to calculate"`
}

func (t CalculatorTool) Schema() *jsonschema.Schema {
	return llms.GenerateSchema[CalculatorToolParams]()
}

func (t CalculatorTool) Execute(ctx context.Context, args []byte) *llms.ToolResult {
	var params CalculatorToolParams
	err := json.Unmarshal(args, &params)
	if err != nil {
		return &llms.ToolResult{
			ID:    "calculator",
			Error: err,
		}
	}

	return &llms.ToolResult{
		ID:      "calculator",
		Content: `Result: 42`, // Simplified for testing
	}
}

