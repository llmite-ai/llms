package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jpoz/llmite"
	"github.com/jpoz/llmite/anthropic"
)

// CalculatorTool implements the Tool interface for basic math operations
type CalculatorTool struct{}

func (t CalculatorTool) Name() string {
	return "calculator"
}

func (t CalculatorTool) Description() string {
	return "Performs basic arithmetic operations. Supports addition, subtraction, multiplication, and division."
}

func (t CalculatorTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "The arithmetic operation to perform",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "The first number",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "The second number",
			},
		},
		"required": []string{"operation", "a", "b"},
	}
}

// WeatherTool implements the Tool interface for weather information
type WeatherTool struct{}

func (t WeatherTool) Name() string {
	return "get_weather"
}

func (t WeatherTool) Description() string {
	return "Get current weather information for a specific location. Returns temperature, conditions, and humidity."
}

func (t WeatherTool) Schema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "The city and state, e.g. San Francisco, CA",
			},
			"unit": map[string]interface{}{
				"type":        "string",
				"description": "Temperature unit",
				"enum":        []string{"celsius", "fahrenheit"},
				"default":     "fahrenheit",
			},
		},
		"required": []string{"location"},
	}
}

// executeCalculator simulates tool execution for the calculator
func executeCalculator(input json.RawMessage) (string, error) {
	var params struct {
		Operation string  `json:"operation"`
		A         float64 `json:"a"`
		B         float64 `json:"b"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse calculator input: %w", err)
	}

	var result float64
	switch params.Operation {
	case "add":
		result = params.A + params.B
	case "subtract":
		result = params.A - params.B
	case "multiply":
		result = params.A * params.B
	case "divide":
		if params.B == 0 {
			return "", fmt.Errorf("division by zero")
		}
		result = params.A / params.B
	default:
		return "", fmt.Errorf("unsupported operation: %s", params.Operation)
	}

	return fmt.Sprintf("%.2f", result), nil
}

// executeWeather simulates tool execution for the weather tool
func executeWeather(input json.RawMessage) (string, error) {
	var params struct {
		Location string `json:"location"`
		Unit     string `json:"unit"`
	}

	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse weather input: %w", err)
	}

	if params.Unit == "" {
		params.Unit = "fahrenheit"
	}

	// Simulate weather data
	temp := "72"
	unit := "°F"
	if params.Unit == "celsius" {
		temp = "22"
		unit = "°C"
	}

	return fmt.Sprintf("Weather in %s: %s%s, partly cloudy, humidity 65%%", params.Location, temp, unit), nil
}

// processToolCalls handles tool execution and returns updated messages
func processToolCalls(messages []llmite.Message, response *llmite.Response) ([]llmite.Message, error) {
	// Add the assistant's response to the conversation
	messages = append(messages, response.Message)

	// Process any tool calls in the response
	for _, part := range response.Message.Parts {
		if toolCall, ok := part.(llmite.ToolCallPart); ok {
			var result string
			var err error

			// Execute the appropriate tool
			switch toolCall.Name {
			case "calculator":
				result, err = executeCalculator(toolCall.Input)
			case "get_weather":
				result, err = executeWeather(toolCall.Input)
			default:
				err = fmt.Errorf("unknown tool: %s", toolCall.Name)
			}

			// Create tool result part
			toolResult := llmite.ToolResultPart{
				ToolCallID: toolCall.ID,
				Name:       toolCall.Name,
				Result:     result,
				Error:      err,
			}

			// Add tool result as a user message
			messages = append(messages, llmite.Message{
				Role:  llmite.RoleUser,
				Parts: []llmite.Part{toolResult},
			})
		}
	}

	return messages, nil
}

func main() {
	// Create tools
	calculator := CalculatorTool{}
	weather := WeatherTool{}

	// Create Anthropic client with tools
	client := anthropic.New(
		anthropic.WithModel("claude-sonnet-4-20250514"),
		anthropic.WithMaxTokens(1000),
		anthropic.WithTools([]llmite.Tool{calculator, weather}),
	)

	// Create conversation with system message
	messages := []llmite.Message{
		llmite.NewTextMessage(llmite.RoleSystem, "You are a helpful assistant with access to calculator and weather tools. When asked to perform calculations or get weather information, use the appropriate tools."),
		llmite.NewTextMessage(llmite.RoleUser, "What's the weather like in San Francisco, CA and what's 25 multiplied by 4?"),
	}

	ctx := context.Background()
	maxTurns := 5 // Prevent infinite loops

	fmt.Println("=== Tool Usage Example ===")
	fmt.Printf("User: %s\n\n", messages[1].Parts[0].(llmite.TextPart).Text)

	// Conversation loop to handle tool calls
	for turn := 0; turn < maxTurns; turn++ {
		// Generate response
		response, err := client.Generate(ctx, messages)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Assistant (Turn %d):\n", turn+1)

		// Display response parts
		hasToolCalls := false
		for i, part := range response.Message.Parts {
			switch p := part.(type) {
			case llmite.TextPart:
				fmt.Printf("  Text: %s\n", p.Text)
			case llmite.ToolCallPart:
				fmt.Printf("  Tool Call %d: %s\n", i, p.Name)
				fmt.Printf("    ID: %s\n", p.ID)
				fmt.Printf("    Input: %s\n", string(p.Input))
				hasToolCalls = true
			default:
				fmt.Printf("  Unknown part type: %T\n", p)
			}
		}

		// If no tool calls, we're done
		if !hasToolCalls {
			break
		}

		// Process tool calls and continue conversation
		messages, err = processToolCalls(messages, response)
		if err != nil {
			log.Fatal(err)
		}

		// Show tool results
		fmt.Printf("Tool Results:\n")
		for _, msg := range messages {
			if msg.Role == llmite.RoleUser {
				for _, part := range msg.Parts {
					if toolResult, ok := part.(llmite.ToolResultPart); ok {
						fmt.Printf("  %s: %s\n", toolResult.Name, toolResult.Result)
						if toolResult.Error != nil {
							fmt.Printf("    Error: %s\n", toolResult.Error)
						}
					}
				}
			}
		}
		fmt.Println()
	}

	fmt.Println("=== Conversation Complete ===")
}

