package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/jpoz/llmite"
)

const ProviderAnthropic = "anthropic"
const Version = "2023-06-01"

type Client struct {
	Model       string
	MaxTokens   int
	Temperature *float64
	TopP        *float64
	TopK        *int
	Tools       []llmite.Tool

	client *http.Client
	token  string
}

type Modifer func(*Client)

// WithApiKey allows you to set the API key on the client. This is useful if you want to
// set the API key directly instead of using the ANTHROPIC_API_KEY or ANTHROPIC_AUTH_TOKEN
// environment variables.
func WithApiKey(key string) Modifer {
	return func(a *Client) {
		a.token = key
	}
}

// WithHttpLogging will log all HTTP requests and responses to the default structured
// logger.
func WithHttpLogging() Modifer {
	return func(a *Client) {
		client := llmite.NewDefaultHTTPClientWithLogging()
		a.client = client
	}
}

// WithModel allows you to set the model on the client.
func WithModel(model string) Modifer {
	return func(a *Client) {
		a.Model = model
	}
}

// WithMaxTokens allows you to set the max tokens on the client.
func WithMaxTokens(maxTokens int) Modifer {
	return func(a *Client) {
		a.MaxTokens = maxTokens
	}
}

// With Tools allows you to set the tools on the client.
func WithTools(tools []llmite.Tool) Modifer {
	return func(a *Client) {
		a.Tools = tools
	}
}

// New creates a new Anthropic client with the packages default options.
// This includes reading the ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, and
// ANTHROPIC_BASE_URL environment variables.
func New(mods ...Modifer) llmite.LLM {
	c := &Client{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1024 * 10,
	}

	for _, mod := range mods {
		mod(c)
	}

	if c.client == nil {
		c.client = llmite.NewHTTPClient(llmite.HTTPClientOptions{})
	}

	if c.token == "" {
		if t := os.Getenv("ANTHROPIC_API_KEY"); t != "" {
			c.token = t
		} else if t := os.Getenv("ANTHROPIC_AUTH_TOKEN"); t != "" {
			c.token = t
		}
	}

	return c
}

func (a *Client) Generate(ctx context.Context, messages []llmite.Message) (*llmite.Response, error) {
	system, requestMessages, err := convertMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to convert messages: %w", err)
	}

	tools, err := convertTools(a.Tools)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to convert tools: %w", err)
	}

	requestBody := CreateMessageRequest{
		Model:     a.Model,
		MaxTokens: a.MaxTokens,
		TopP:      a.TopP,
		TopK:      a.TopK,
		Messages:  requestMessages,
	}

	if len(system) > 0 {
		requestBody.System = system
	}

	if len(tools) > 0 {
		requestBody.Tools = tools
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to marshal request body: %w", err)
	}

	bodyBuffer := io.NopCloser(bytes.NewReader(bodyBytes))

	request, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bodyBuffer)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("anthropic-version", Version)
	request.Header.Set("x-api-key", a.token)

	response, err := a.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		b, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("anthropic: request failed with status %d: %s", response.StatusCode, string(b))
	}

	var responseBody CreateMessageResponse
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return nil, fmt.Errorf("anthropic: failed to decode response body: %w", err)
	}

	parts := make([]llmite.Part, 0, len(responseBody.Content))

	for _, block := range responseBody.Content {
		switch b := block.(type) {
		case ResponseTextBlock:
			parts = append(parts, llmite.TextPart{Text: b.Text})
		case ResponseToolUseBlock:
			parts = append(parts, llmite.ToolCallPart{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input, // Anthropic does not return input in response
			})
		default:
			return nil, fmt.Errorf("anthropic: unsupported response content block type: %T", b)
		}
	}
	out := &llmite.Response{
		ID: responseBody.ID,
		Message: llmite.Message{
			Role:  llmite.RoleAssistant,
			Parts: parts,
		},
		Provider: ProviderAnthropic,
		Raw:      responseBody,
	}

	return out, nil
}

func (a *Client) GenerateStream(ctx context.Context, messages []llmite.Message, fn llmite.StreamFunc) (*llmite.Response, error) {
	return nil, fmt.Errorf("anthropic: streaming is not supported")
}

func convertMessages(messages []llmite.Message) (SystemPrompt, []InputMessage, error) {
	system := make(SystemPrompt, 0)
	out := make([]InputMessage, 0, len(messages))

	for _, msg := range messages {
		if msg.Role == llmite.RoleSystem {
			for _, part := range msg.Parts {
				switch p := part.(type) {
				case llmite.TextPart:
					block := RequestTextBlock{
						Text: p.Text,
					}
					system = append(system, block)
				default:
					return nil, nil, fmt.Errorf("anthropic: unsupported system message part type: %T", p)
				}
			}
			continue
		}
		inputMsg := InputMessage{
			Content: make([]ContentBlock, 0, len(msg.Parts)),
		}

		switch msg.Role {
		case llmite.RoleUser:
			inputMsg.Role = "user"
		case llmite.RoleAssistant:
			inputMsg.Role = "assistant"
		default:
			return nil, nil, fmt.Errorf("anthropic: unsupported message role: %s", msg.Role)
		}

		for _, part := range msg.Parts {
			switch p := part.(type) {
			case llmite.TextPart:
				block := RequestTextBlock{
					Text: p.Text,
				}
				inputMsg.Content = append(inputMsg.Content, block)
			case llmite.ToolCallPart:
				block := RequestToolUseBlock{
					ID:   p.ID,
					Name: p.Name,
					// Input: p.Input,
				}
				inputMsg.Content = append(inputMsg.Content, block)
			case llmite.ToolResultPart:
				block := RequestToolResultBlock{
					ToolUseID: p.ToolCallID,
					// Content: []RequestToolResultContentBlock{
					// 	{Text: p.Result},
					// },
				}
				inputMsg.Content = append(inputMsg.Content, block)
			default:
				return nil, nil, fmt.Errorf("anthropic: unsupported message part type: %T", p)
			}
		}

		out = append(out, inputMsg)
	}

	return system, out, nil
}

func convertTools(tools []llmite.Tool) ([]Tool, error) {
	out := make([]Tool, 0, len(tools))

	for _, tool := range tools {
		anthTool := Tool{
			Name: tool.Name(),
		}

		if desc := tool.Description(); desc != "" {
			anthTool.Description = &desc
		}

		schema := tool.Schema()
		if schema != nil {
			properties, ok := schema["properties"].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("anthropic: tool schema 'properties' is not a map[string]interface{}")
			}

			required, ok := schema["required"].([]string)
			if !ok {
				required = []string{}
			}

			anthTool.InputSchema = InputSchema{
				Properties: properties,
				Required:   required,
			}
		}

		out = append(out, anthTool)
	}

	return out, nil
}
