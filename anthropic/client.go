package anthropic

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"

	"github.com/llmite-ai/llms"
)

const ProviderAnthropic = "anthropic"

type Client struct {
	Model       string
	MaxTokens   int64
	Temperature *float64
	TopP        *float64
	TopK        *int64
	Tools       []llms.Tool

	client  *anthropic.Client
	options []option.RequestOption
}

type Modifer func(*Client)

// WithOptions allows you to set options on the client. This is useful for setting
// options that are not exposed by the llms.Anthropic struct, such as setting a
// custom HTTP client, timeout, or base URL.
func WithAnthropicClientOptions(options ...option.RequestOption) Modifer {
	return func(a *Client) {
		a.options = options
	}
}

// WithHttpLogging will log all HTTP requests and responses to the default structured
// logger.
func WithHttpLogging() Modifer {
	return func(a *Client) {
		client := llms.NewDefaultHTTPClientWithLogging()
		a.options = append(a.options, option.WithHTTPClient(client))
	}
}

// WithModel allows you to set the model on the client.
func WithModel(model string) Modifer {
	return func(a *Client) {
		a.Model = model
	}
}

// WithMaxTokens allows you to set the max tokens on the client.
func WithMaxTokens(maxTokens int64) Modifer {
	return func(a *Client) {
		a.MaxTokens = maxTokens
	}
}

// With Tools allows you to set the tools on the client.
func WithTools(tools []llms.Tool) Modifer {
	return func(a *Client) {
		a.Tools = tools
	}
}

// New creates a new Anthropic client with the packages default options.
// This includes reading the ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, and
// ANTHROPIC_BASE_URL environment variables.
func New(mods ...Modifer) llms.LLM {
	c := &Client{
		Model:     string(anthropic.ModelClaude3_7SonnetLatest),
		MaxTokens: 1024,
		options:   []option.RequestOption{},
	}

	for _, mod := range mods {
		mod(c)
	}

	if c.client == nil {
		ac := anthropic.NewClient(c.options...)
		c.client = &ac
	}

	return c
}

// GetClient returns the underlying anthropic client.
func (a *Client) GetClient() *anthropic.Client {
	return a.client
}

func (a *Client) BuildRequest(ctx context.Context, messages []llms.Message) (*anthropic.MessageNewParams, error) {
	system, anthMessages, err := convertMessages(messages)
	if err != nil {
		return nil, err
	}

	tools, err := convertTools(a.Tools)
	if err != nil {
		return nil, err
	}

	body := anthropic.MessageNewParams{
		MaxTokens: a.MaxTokens,
		Model:     anthropic.Model(a.Model),
		Messages:  anthMessages,
		System:    system,
		Tools:     tools,
	}

	if a.Temperature != nil {
		body.Temperature = param.NewOpt(*a.Temperature)
	}

	return &body, nil
}

func (a *Client) Generate(ctx context.Context, messages []llms.Message) (*llms.Response, error) {
	body, err := a.BuildRequest(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to build request: %w", err)
	}

	msg, err := a.client.Messages.New(
		ctx,
		*body,
	)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to generate message: %w", err)
	}

	return convertMessageToResponse(msg)
}

func (a *Client) GenerateStream(ctx context.Context, messages []llms.Message, fn llms.StreamFunc) (*llms.Response, error) {
	body, err := a.BuildRequest(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to build request: %w", err)
	}

	stream := a.client.Messages.NewStreaming(
		ctx,
		*body,
	)
	if stream == nil {
		return nil, fmt.Errorf("anthropic: failed to create streaming request")
	}

	message := &anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			if !fn(nil, fmt.Errorf("anthropic: failed to accumulate message: %w", err)) {
				return nil, fmt.Errorf("anthropic: failed to accumulate message: %w", err)
			}
			continue
		}

		response, err := convertMessageToResponse(message)
		if err != nil {
			if !fn(nil, fmt.Errorf("anthropic: failed to convert message to response: %w", err)) {
				return nil, fmt.Errorf("anthropic: failed to convert message to response: %w", err)
			}
			continue
		}

		if !fn(response, nil) {
			return response, nil
		}
	}

	if stream.Err() != nil {
		return nil, fmt.Errorf("anthropic: streaming request failed: %w", stream.Err())
	}

	return convertMessageToResponse(message)
}

func convertMessageToResponse(msg *anthropic.Message) (*llms.Response, error) {
	msgOut := llms.Message{
		Role:  llms.RoleAssistant,
		Parts: []llms.Part{},
	}

	errs := make([]error, 0)

	for i, block := range msg.Content {
		switch block.Type {
		case "text":
			msgOut.Parts = append(msgOut.Parts, llms.TextPart{
				Text: block.Text,
			})
		case "tool_use":
			msgOut.Parts = append(msgOut.Parts, llms.ToolCallPart{
				ID:    block.ToolUseID,
				Name:  block.Name,
				Input: block.Input,
			})
		default:
			errs = append(errs, fmt.Errorf("anthropic: unsupported content block type at index %d: %v", i, block))
		}
	}

	out := &llms.Response{
		ID:       msg.ID,
		Message:  msgOut,
		Provider: ProviderAnthropic,
		Raw:      msg,
	}

	if len(errs) > 0 {
		return out, errors.Join(errs...)
	}

	return out, nil
}

func convertMessages(messages []llms.Message) ([]anthropic.TextBlockParam, []anthropic.MessageParam, error) {
	system := []anthropic.TextBlockParam{}
	out := make([]anthropic.MessageParam, 0, len(messages))

	for i, message := range messages {
		// If the message is a system message, we need to convert it to a
		// system prompt. We do this by appending the message parts to the
		// system TextBlockParams.
		if message.Role == llms.RoleSystem {
			for _, part := range message.Parts {
				switch p := part.(type) {
				case llms.TextPart:
					system = append(system, anthropic.TextBlockParam{
						Text: p.Text,
					})
				default:
					return system, nil, fmt.Errorf("[message %d] anthropic: unsupported message part type: %T", i, p)
				}
			}

			continue
		}

		// Convert the role
		anthMessage := anthropic.MessageParam{}
		switch message.Role {
		case llms.RoleUser:
			anthMessage.Role = anthropic.MessageParamRoleUser
		case llms.RoleAssistant:
			anthMessage.Role = anthropic.MessageParamRoleAssistant
		default:
			return system, nil, fmt.Errorf("[message %d] anthropic: unsupported message role: %s", i, message.Role)
		}

		// Convert the message parts
		anthMessage.Content = []anthropic.ContentBlockParamUnion{}
		for j, part := range message.Parts {
			switch p := part.(type) {
			case llms.TextPart:
				anthMessage.Content = append(anthMessage.Content, anthropic.ContentBlockParamUnion{
					OfText: &anthropic.TextBlockParam{
						Text: p.Text,
					},
				})
			case llms.ToolCallPart:
				anthMessage.Content = append(anthMessage.Content, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    p.ID,
						Name:  p.Name,
						Input: p.Input,
					},
				})
			case llms.ToolResultPart:
				c := anthropic.ContentBlockParamUnion{
					OfToolResult: &anthropic.ToolResultBlockParam{
						ToolUseID: p.ToolCallID,
						Content: []anthropic.ToolResultBlockParamContentUnion{
							{
								OfText: &anthropic.TextBlockParam{
									Text: p.Result,
								},
							},
						},
					},
				}

				if p.Error != nil {
					c.OfToolResult.IsError = param.NewOpt(true)
				}

				anthMessage.Content = append(anthMessage.Content, c)
			default:
				return system, nil, fmt.Errorf("[message %d, part %d] anthropic: unsupported message part type: %T", i, j, p)
			}
		}

		out = append(out, anthMessage)
	}

	return system, out, nil
}

func convertTools(tools []llms.Tool) ([]anthropic.ToolUnionParam, error) {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))

	for _, tool := range tools {
		anthTool := anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name(),
				Description: param.NewOpt(tool.Description()),
			},
		}

		schema := tool.Schema()

		anthTool.OfTool.InputSchema = anthropic.ToolInputSchemaParam{
			Properties: schema.Properties,
			Required:   schema.Required,
		}

		out = append(out, anthTool)
	}

	return out, nil
}
