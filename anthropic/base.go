package anthropic

import (
	"context"
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"

	"github.com/jpoz/llmite"
)

const ProviderAnthropic = "anthropic"

type Anthropic struct {

	// Settings
	Model       string
	MaxTokens   int64
	Temperature *float64
	TopP        *float64
	TopK        *int64
	Tools       []llmite.Tool

	client  *anthropic.Client
	options []option.RequestOption
}

type Modifer func(*Anthropic)

// WithOptions allows you to set options on the client. This is useful for setting
// options that are not exposed by the llmite.Anthropic struct, such as setting a
// custom HTTP client, timeout, or base URL.
func WithAnthropicClientOptions(options ...option.RequestOption) Modifer {
	return func(a *Anthropic) {
		a.options = options
	}
}

// WithHttpLogging will log all HTTP requests and responses to the default structured
// logger.
func WithHttpLogging() Modifer {
	return func(a *Anthropic) {
		client := llmite.NewDefaultHTTPClientWithLogging()
		a.options = append(a.options, option.WithHTTPClient(client))
	}
}

// WithModel allows you to set the model on the client.
func WithModel(model string) Modifer {
	return func(a *Anthropic) {
		a.Model = model
	}
}

// WithMaxTokens allows you to set the max tokens on the client.
func WithMaxTokens(maxTokens int64) Modifer {
	return func(a *Anthropic) {
		a.MaxTokens = maxTokens
	}
}

// With Tools allows you to set the tools on the client.
func WithTools(tools []llmite.Tool) Modifer {
	return func(a *Anthropic) {
		a.Tools = tools
	}
}

// NewAnthropicFrom creates a new Anthropic client with the packages default options.
// This includes reading the ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, and
// ANTHROPIC_BASE_URL environment variables.
func NewAnthropic(mods ...Modifer) *Anthropic {
	c := &Anthropic{
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
func (a *Anthropic) GetClient() *anthropic.Client {
	return a.client
}

func (a *Anthropic) Generate(ctx context.Context, messages []llmite.Message) (*llmite.Response, error) {
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

	anthResponse, err := a.client.Messages.New(
		ctx,
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to generate message: %w", err)
	}

	msgOut := llmite.Message{
		Role:  llmite.RoleAssistant,
		Parts: []llmite.Part{},
	}

	errs := make([]error, 0)

	for i, block := range anthResponse.Content {
		switch block.Type {
		case "text":
			msgOut.Parts = append(msgOut.Parts, llmite.TextPart{
				Text: block.Text,
			})
		case "tool_use":
			msgOut.Parts = append(msgOut.Parts, llmite.ToolCallPart{
				ID:    block.ToolUseID,
				Name:  block.Name,
				Input: block.Input,
			})
		default:
			errs = append(errs, fmt.Errorf("anthropic: unsupported content block type at index %d: %v", i, block))
		}
	}

	out := &llmite.Response{
		ID:       anthResponse.ID,
		Message:  msgOut,
		Provider: ProviderAnthropic,
		Raw:      anthResponse,
	}

	if len(errs) > 0 {
		return out, errors.Join(errs...)
	}

	return out, nil
}

func (a *Anthropic) GenerateStream(ctx context.Context, messages []llmite.Message) (*llmite.Response, error) {
	return nil, fmt.Errorf("anthropic: streaming is not supported")
}

func (a *Anthropic) copy() *Anthropic {
	return &Anthropic{
		client:      a.client,
		MaxTokens:   a.MaxTokens,
		Temperature: a.Temperature,
		Model:       a.Model,
		TopP:        a.TopP,
		TopK:        a.TopK,
		Tools:       a.Tools,
	}
}

func convertMessages(messages []llmite.Message) ([]anthropic.TextBlockParam, []anthropic.MessageParam, error) {
	system := []anthropic.TextBlockParam{}
	out := make([]anthropic.MessageParam, 0, len(messages))

	for i, message := range messages {
		// If the message is a system message, we need to convert it to a
		// system prompt. We do this by appending the message parts to the
		// system TextBlockParams.
		if message.Role == llmite.RoleSystem {
			for _, part := range message.Parts {
				switch p := part.(type) {
				case llmite.TextPart:
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
		case llmite.RoleUser:
			anthMessage.Role = anthropic.MessageParamRoleUser
		case llmite.RoleAssistant:
			anthMessage.Role = anthropic.MessageParamRoleAssistant
		default:
			return system, nil, fmt.Errorf("[message %d] anthropic: unsupported message role: %s", i, message.Role)
		}

		// Convert the message parts
		anthMessage.Content = []anthropic.ContentBlockParamUnion{}
		for j, part := range message.Parts {
			switch p := part.(type) {
			case llmite.TextPart:
				anthMessage.Content = append(anthMessage.Content, anthropic.ContentBlockParamUnion{
					OfText: &anthropic.TextBlockParam{
						Text: p.Text,
					},
				})
			case llmite.ToolCallPart:
				anthMessage.Content = append(anthMessage.Content, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    p.ID,
						Name:  p.Name,
						Input: p.Input,
					},
				})
			case llmite.ToolResultPart:
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

func convertTools(tools []llmite.Tool) ([]anthropic.ToolUnionParam, error) {
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
