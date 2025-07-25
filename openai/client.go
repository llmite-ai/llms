package openai

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/llmite-ai/llms"
)

const ProviderOpenAI = "openai"

type Client struct {
	Model       string
	MaxTokens   int64
	Temperature *float64
	TopP        *float64
	Tools       []llmite.Tool

	client  *openai.Client
	options []option.RequestOption
}

type Modifier func(*Client)

// WithOpenAIClientOptions allows you to set options on the client. This is useful for setting
// options that are not exposed by the llmite.OpenAI struct, such as setting a
// custom HTTP client, timeout, or base URL.
func WithOpenAIClientOptions(options ...option.RequestOption) Modifier {
	return func(c *Client) {
		c.options = options
	}
}

// WithHttpLogging will log all HTTP requests and responses to the default structured
// logger.
func WithHttpLogging() Modifier {
	return func(c *Client) {
		client := llmite.NewDefaultHTTPClientWithLogging()
		c.options = append(c.options, option.WithHTTPClient(client))
	}
}

// WithModel allows you to set the model on the client. The default model is "gpt-4o".
func WithModel(model string) Modifier {
	return func(c *Client) {
		c.Model = model
	}
}

// WithMaxTokens allows you to set the max tokens on the client.
func WithMaxTokens(maxTokens int64) Modifier {
	return func(c *Client) {
		c.MaxTokens = maxTokens
	}
}

// WithTemperature allows you to set the temperature on the client.
func WithTemperature(temperature float64) Modifier {
	return func(c *Client) {
		c.Temperature = &temperature
	}
}

// WithTopP allows you to set the top_p on the client.
func WithTopP(topP float64) Modifier {
	return func(c *Client) {
		c.TopP = &topP
	}
}

// WithTools allows you to set the tools on the client.
func WithTools(tools []llmite.Tool) Modifier {
	return func(c *Client) {
		c.Tools = tools
	}
}

// New creates a new OpenAI client with the default options.
// This includes reading the OPENAI_API_KEY environment variable.
func New(mods ...Modifier) llmite.LLM {
	c := &Client{
		Model:     "gpt-4o",
		MaxTokens: 1024,
		options:   []option.RequestOption{},
	}

	for _, mod := range mods {
		mod(c)
	}

	if c.client == nil {
		client := openai.NewClient(c.options...)
		c.client = &client
	}

	return c
}

// GetClient returns the underlying OpenAI client.
func (c *Client) GetClient() *openai.Client {
	return c.client
}

func (c *Client) Generate(ctx context.Context, messages []llmite.Message) (*llmite.Response, error) {
	oaiMessages, err := convertMessages(messages)
	if err != nil {
		return nil, err
	}

	tools, err := convertTools(c.Tools)
	if err != nil {
		return nil, err
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(c.Model),
		Messages: oaiMessages,
		Tools:    tools,
	}

	if c.MaxTokens > 0 {
		params.MaxTokens = openai.Int(c.MaxTokens)
	}

	if c.Temperature != nil {
		params.Temperature = openai.Float(*c.Temperature)
	}

	if c.TopP != nil {
		params.TopP = openai.Float(*c.TopP)
	}

	oaiResponse, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai: failed to generate message: %w", err)
	}

	if len(oaiResponse.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices returned")
	}

	choice := oaiResponse.Choices[0]
	msgOut := llmite.Message{
		Role:  llmite.RoleAssistant,
		Parts: []llmite.Part{},
	}

	errs := make([]error, 0)

	// Handle text content
	if choice.Message.Content != "" {
		msgOut.Parts = append(msgOut.Parts, llmite.TextPart{
			Text: choice.Message.Content,
		})
	}

	// Handle tool calls
	for _, toolCall := range choice.Message.ToolCalls {
		if toolCall.Type == "function" {
			msgOut.Parts = append(msgOut.Parts, llmite.ToolCallPart{
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: []byte(toolCall.Function.Arguments),
			})
		}
	}

	out := &llmite.Response{
		ID:       oaiResponse.ID,
		Message:  msgOut,
		Provider: ProviderOpenAI,
		Raw:      oaiResponse,
	}

	if len(errs) > 0 {
		return out, errors.Join(errs...)
	}

	return out, nil
}

func (c *Client) GenerateStream(ctx context.Context, messages []llmite.Message, fn llmite.StreamFunc) (*llmite.Response, error) {
	oaiMessages, err := convertMessages(messages)
	if err != nil {
		return nil, err
	}

	tools, err := convertTools(c.Tools)
	if err != nil {
		return nil, err
	}

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(c.Model),
		Messages: oaiMessages,
		Tools:    tools,
	}

	if c.MaxTokens > 0 {
		params.MaxTokens = openai.Int(c.MaxTokens)
	}

	if c.Temperature != nil {
		params.Temperature = openai.Float(*c.Temperature)
	}

	if c.TopP != nil {
		params.TopP = openai.Float(*c.TopP)
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)
	defer stream.Close()

	out := &llmite.Response{
		Message: llmite.Message{
			Role:  llmite.RoleAssistant,
			Parts: []llmite.Part{},
		},
		Provider: ProviderOpenAI,
	}

	// Track tool calls across chunks
	toolCalls := make(map[string]*llmite.ToolCallPart)

	for stream.Next() {
		chunk := stream.Current()
		
		if chunk.ID != "" && out.ID == "" {
			out.ID = chunk.ID
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		// Handle text content
		if delta.Content != "" {
			textPart := llmite.TextPart{Text: delta.Content}
			out.Message.Parts = append(out.Message.Parts, textPart)
			
			// Call the stream function with the response and error
			continueStream := fn(out, nil)
			if !continueStream {
				return out, nil
			}
		}

		// Handle tool calls
		for _, toolCall := range delta.ToolCalls {
			if toolCall.Type == "function" {
				if existingCall, exists := toolCalls[toolCall.ID]; exists {
					// Append to existing tool call
					existingCall.Input = append(existingCall.Input, []byte(toolCall.Function.Arguments)...)
				} else {
					// Create new tool call
					newCall := &llmite.ToolCallPart{
						ID:    toolCall.ID,
						Name:  toolCall.Function.Name,
						Input: []byte(toolCall.Function.Arguments),
					}
					toolCalls[toolCall.ID] = newCall
				}
			}
		}

		out.Raw = chunk
	}

	if err := stream.Err(); err != nil {
		return out, fmt.Errorf("openai: streaming error: %w", err)
	}

	// Add accumulated tool calls to the message
	for _, toolCall := range toolCalls {
		out.Message.Parts = append(out.Message.Parts, *toolCall)
	}

	return out, nil
}

func convertMessages(messages []llmite.Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for i, message := range messages {
		switch message.Role {
		case llmite.RoleSystem:
			// Convert system message
			content := ""
			for _, part := range message.Parts {
				switch p := part.(type) {
				case llmite.TextPart:
					content += p.Text
				default:
					return nil, fmt.Errorf("[message %d] openai: unsupported system message part type: %T", i, p)
				}
			}
			
			out = append(out, openai.SystemMessage(content))

		case llmite.RoleUser:
			// Convert user message
			content := ""
			for _, part := range message.Parts {
				switch p := part.(type) {
				case llmite.TextPart:
					content += p.Text
				default:
					return nil, fmt.Errorf("[message %d] openai: unsupported user message part type: %T", i, p)
				}
			}
			
			out = append(out, openai.UserMessage(content))

		case llmite.RoleAssistant:
			// Convert assistant message
			content := ""
			hasToolResults := false
			
			for _, part := range message.Parts {
				switch p := part.(type) {
				case llmite.TextPart:
					content += p.Text
				case llmite.ToolCallPart:
					// TODO: Handle tool calls properly
				case llmite.ToolResultPart:
					hasToolResults = true
					// Tool results are handled as separate messages
					out = append(out, openai.ToolMessage(p.Result, p.ToolCallID))
				default:
					return nil, fmt.Errorf("[message %d] openai: unsupported assistant message part type: %T", i, p)
				}
			}
			
			// Only add assistant message if there's content and no tool results
			if content != "" && !hasToolResults {
				out = append(out, openai.AssistantMessage(content))
			}
			
			// TODO: Implement tool calls - this is complex due to the OpenAI SDK API structure

		default:
			return nil, fmt.Errorf("[message %d] openai: unsupported message role: %s", i, message.Role)
		}
	}

	return out, nil
}

func convertTools(tools []llmite.Tool) ([]openai.ChatCompletionToolParam, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	out := make([]openai.ChatCompletionToolParam, 0, len(tools))

	for _, tool := range tools {
		schema := tool.Schema()
		if schema == nil {
			return nil, fmt.Errorf("openai: tool %s has no schema", tool.Name())
		}

		// Convert jsonschema to map[string]interface{}
		schemaMap := make(map[string]interface{})
		if schema.Type != "" {
			schemaMap["type"] = schema.Type
		}
		if schema.Properties != nil {
			schemaMap["properties"] = schema.Properties
		}
		if schema.Required != nil {
			schemaMap["required"] = schema.Required
		}
		if schema.Description != "" {
			schemaMap["description"] = schema.Description
		}

		out = append(out, openai.ChatCompletionToolParam{
			Type: "function",
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name(),
				Description: openai.String(tool.Description()),
				Parameters:  openai.FunctionParameters(schemaMap),
			},
		})
	}

	return out, nil
}