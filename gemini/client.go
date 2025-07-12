package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"

	"github.com/google/uuid"
	"github.com/jpoz/llmite"
)

const ProviderGemini = "gemini"

type Client struct {
	Model              string
	MaxTokens          int64
	Temperature        *float64
	TopP               *float64
	TopK               *int64
	Tools              []llmite.Tool
	SystemInstructions []llmite.Part

	client *genai.Client
	config *genai.ClientConfig
}

type Modifer func(*Client)

// WithGeminiClient allows you to set a custom Gemini client. This is useful if you want to
// share a single client across multiple llmite clients, or if you want to customize the
// underlying client in a way that is not supported by llmite.
func WithGeminiClient(client *genai.Client) Modifer {
	return func(c *Client) {
		c.client = client
	}
}

// WithApiKey allows you to set the API key on the client.
func WithApiKey(apiKey string) Modifer {
	return func(c *Client) {
		if c.config == nil {
			c.config = &genai.ClientConfig{}
		}
		c.config.APIKey = apiKey
	}
}

// WithModel allows you to set the model on the client. The default model is "gemini-2.5-pro-preview-06-05".
func WithModel(model string) Modifer {
	return func(c *Client) {
		c.Model = model
	}
}

// WithSystemInstructions allows you to set system instructions on the client. These instructions will be prepended to every request.
func WithSystemInstructions(parts ...llmite.Part) Modifer {
	return func(c *Client) {
		c.SystemInstructions = append(c.SystemInstructions, parts...)
	}
}

// WithSystemInstruction is a convenience function to add a single text system instruction to the client. This is useful for setting a system prompt that is a single string.
func WithSystemInstruction(prompt string) Modifer {
	parts := []llmite.Part{llmite.TextPart{Text: prompt}}

	return func(c *Client) {
		c.SystemInstructions = append(c.SystemInstructions, parts...)
	}
}

// WithTools allows you to set tools on the client. These tools will be available for function calling.
func WithTools(tools []llmite.Tool) Modifer {
	return func(c *Client) {
		c.Tools = tools
	}
}

// WithHttpLogging will log all HTTP requests and responses to the default structured logger.
func WithHttpLogging() Modifer {
	return func(c *Client) {
		if c.config == nil {
			c.config = &genai.ClientConfig{}
		}
		client := llmite.NewDefaultHTTPClientWithLogging()
		c.config.HTTPClient = client
	}
}

// New creates a new Gemini client. You can pass in modifiers to customize the client.
//
//   - Environment Variables for BackendGeminiAPI:
//
//   - GEMINI_API_KEY: Specifies the API key for the Gemini API.
//
//   - GOOGLE_API_KEY: Can also be used to specify the API key for the Gemini API.
//     If both GOOGLE_API_KEY and GEMINI_API_KEY are set, GOOGLE_API_KEY will be used.
//
//   - Environment Variables for BackendVertexAI:
//
//   - GOOGLE_GENAI_USE_VERTEXAI: Must be set to "1" or "true" to use the Vertex AI
//     backend.
//
//   - GOOGLE_CLOUD_PROJECT: Required. Specifies the GCP project ID.
//
//   - GOOGLE_CLOUD_LOCATION or GOOGLE_CLOUD_REGION: Required. Specifies the GCP
//     location/region.
func New(modifiers ...Modifer) (llmite.LLM, error) {
	c := &Client{
		Model:  "gemini-2.5-pro-preview-06-05",
		config: &genai.ClientConfig{},
	}
	for _, mod := range modifiers {
		mod(c)
	}

	if c.client == nil {
		client, err := genai.NewClient(nil, c.config)
		if err != nil {
			return nil, err
		}

		c.client = client
	}

	return c, nil
}

func (c *Client) Generate(ctx context.Context, messages []llmite.Message) (*llmite.Response, error) {
	return nil, fmt.Errorf("GenerateResponse not implemented for GeminiProvider")
}

func (c *Client) GenerateStream(ctx context.Context, messages []llmite.Message, fn llmite.StreamFunc) (*llmite.Response, error) {
	config := &genai.GenerateContentConfig{}
	contents := make([]*genai.Content, 0, len(messages))

	if c.SystemInstructions != nil && len(c.SystemInstructions) > 0 {
		parts := []*genai.Part{}
		for _, p := range c.SystemInstructions {
			switch part := p.(type) {
			case llmite.TextPart:
				parts = append(parts, &genai.Part{Text: part.Text})
			default:
				return nil, fmt.Errorf("unsupported system instruction part type for Gemini: %T", part)
			}
		}

		config.SystemInstruction = &genai.Content{
			Parts: parts,
		}
	}

	// Configure tools if available
	if c.Tools != nil && len(c.Tools) > 0 {
		tools := genai.Tool{
			FunctionDeclarations: make([]*genai.FunctionDeclaration, 0, len(c.Tools)),
		}
		for _, tool := range c.Tools {
			funcDef := &genai.FunctionDeclaration{
				Name:        tool.Name(),
				Description: tool.Description(),
			}

			// Convert tool schema to JSON for Gemini
			if schema := tool.Schema(); schema != nil {
				// schemaBytes, err := json.Marshal(schema)
				// if err != nil {
				// 	return nil, fmt.Errorf("failed to marshal tool schema for %s: %w", tool.Name(), err)
				// }
				funcDef.ParametersJsonSchema = schema
			}

			tools.FunctionDeclarations = append(tools.FunctionDeclarations, funcDef)
		}
		config.Tools = []*genai.Tool{&tools}
	}

	for _, msg := range messages {
		parts := []*genai.Part{}

		for _, p := range msg.Parts {
			switch part := p.(type) {
			case llmite.TextPart:
				parts = append(parts, &genai.Part{Text: part.Text})
			case llmite.ToolCallPart:
				// parts = append(parts, &genai.Part{FunctionCall: &genai.FunctionCall{
				// 	ID:   part.ID,
				// 	Name: part.Name,
				// 	Args: part.InputString(),
				// }})
			case llmite.ToolResultPart:
				parts = append(parts, &genai.Part{FunctionResponse: &genai.FunctionResponse{
					ID:       part.ToolCallID,
					Name:     part.Name,
					Response: map[string]any{"content": part.Result},
				}})
			}
		}

		content := &genai.Content{
			Parts: parts,
		}

		switch msg.Role {
		case llmite.RoleUser:
			content.Role = genai.RoleUser
		case llmite.RoleAssistant:
			content.Role = genai.RoleModel
		default:
			return nil, fmt.Errorf("unsupported message role for Gemini: %q", msg.Role)
		}

		contents = append(contents, content)
	}

	stream := c.client.Models.GenerateContentStream(
		ctx, c.Model, contents, config)

	out := llmite.Response{}
	for resp, err := range stream {
		fmt.Printf("Gemini response: %+v, error: %v\n", resp, err)
		if err != nil {
			return nil, err
		}

		if len(resp.Candidates) > 0 {
			candidate := resp.Candidates[0]
			if candidate.Content != nil {
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						out.Message.Parts = append(out.Message.Parts, llmite.TextPart{Text: part.Text})
					}
					if part.FunctionCall != nil {
						id := part.FunctionCall.ID
						if id == "" {
							id = fmt.Sprintf("call-%s", uuid.NewString())
						}
						bts, err := json.Marshal(part.FunctionCall.Args)
						if err != nil {
							return nil, fmt.Errorf("failed to marshal Gemini function call args: %v -> %w", part.FunctionCall.Args, err)
						}

						out.Message.Parts = append(out.Message.Parts, llmite.ToolCallPart{
							ID:    id,
							Name:  part.FunctionCall.Name,
							Input: bts,
						})
					}
				}
			}
		}

		out.Raw = resp

		fn(&out, err)
	}

	return &out, nil
}

//
// func (gp *GeminiProvider) GenerateStreamResponse(ctx context.Context, messages []*types.Message, callback func(string)) (*LLMResponse, error) {
// 	config := &genai.GenerateContentConfig{}
// 	contents := make([]*genai.Content, 0, len(messages))
// 	for _, msg := range messages {
// 		parts := []*genai.Part{}
//
// 		if msg.Content != "" {
// 			parts = append(parts, &genai.Part{
// 				Text: msg.Content,
// 			})
// 		}
//
// 		if msg.ToolCalls != nil {
// 			for _, toolCall := range msg.ToolCalls {
// 				parts = append(parts, &genai.Part{
// 					FunctionCall: &genai.FunctionCall{
// 						ID:   toolCall.ID,
// 						Name: toolCall.Name,
// 						Args: toolCall.Arguments,
// 					}})
// 			}
// 		}
//
// 		if msg.ToolResults != nil {
// 			for _, toolResult := range msg.ToolResults {
// 				parts = append(parts, &genai.Part{
// 					FunctionResponse: &genai.FunctionResponse{
// 						ID:   toolResult.ID,
// 						Name: toolResult.Name,
// 						Response: map[string]any{
// 							"content": toolResult.Content, // TODO: Handle more complex responses
// 						},
// 					}})
// 			}
// 		}
//
// 		role := genai.RoleModel
// 		if msg.Role == types.RoleUser {
// 			role = genai.RoleUser
// 		}
// 		if msg.Role == types.RoleSystem {
// 			if config.SystemInstruction == nil {
// 				config.SystemInstruction = &genai.Content{Parts: []*genai.Part{}}
// 			}
// 			config.SystemInstruction.Parts = append(config.SystemInstruction.Parts, parts...)
// 			continue
// 		}
//
// 		contents = append(contents, &genai.Content{
// 			Role:  role,
// 			Parts: parts,
// 		})
// 	}
//
// 	toolList := gp.toolRegistry.List()
// 	if len(toolList) > 0 {
// 		tools := genai.Tool{
// 			FunctionDeclarations: make([]*genai.FunctionDeclaration, 0, len(toolList)),
// 		}
// 		for i, toolInfo := range toolList {
// 			funcDef := &genai.FunctionDeclaration{
// 				Behavior:             genai.BehaviorBlocking,
// 				Description:          toolInfo.Description,
// 				Name:                 toolInfo.Name,
// 				ParametersJsonSchema: toolInfo.Schema,
// 			}
//
// 			slog.Info("Registering tool for Gemini",
// 				"name", funcDef.Name,
// 				"index", i,
// 			)
//
// 			tools.FunctionDeclarations = append(tools.FunctionDeclarations, funcDef)
// 		}
// 		config.Tools = []*genai.Tool{&tools}
// 	}
// 	inspect, _ := json.MarshalIndent(contents, "", "  ")
// 	slog.Info("Generating content stream with Gemini",
// 		"model", gp.model,
// 		"contents", inspect,
// 		"config", config)
//
// 	stream := gp.client.Models.GenerateContentStream(
// 		ctx, gp.model, contents, config)
//
// 	llmResponse := &LLMResponse{
// 		Usage:     types.TokenUsage{},
// 		ModelUsed: gp.model,
// 	}
//
// 	for response, err := range stream {
// 		inspect, _ := json.MarshalIndent(response, "", "  ")
// 		slog.Info("Received response from Gemini stream",
// 			"response", inspect,
// 			"error", err)
//
// 		if err != nil {
// 			slog.Error("Error in Gemini stream", "error", err)
// 			return nil, err
// 		}
//
// 		if len(response.Candidates) == 0 {
// 			continue
// 		}
//
// 		if response.UsageMetadata != nil {
// 			llmResponse.Usage.PromptTokens += int(response.UsageMetadata.PromptTokenCount)
// 			llmResponse.Usage.CompletionTokens += int(response.UsageMetadata.CandidatesTokenCount) +
// 				int(response.UsageMetadata.ToolUsePromptTokenCount) + int(response.UsageMetadata.ThoughtsTokenCount) +
// 				int(response.UsageMetadata.CachedContentTokenCount)
// 			llmResponse.Usage.TotalTokens = llmResponse.Usage.PromptTokens + llmResponse.Usage.CompletionTokens
// 		}
//
// 		candidate := response.Candidates[0]
// 		if candidate.Content == nil {
// 			continue
// 		}
//
// 		// Process each part of the response
// 		for _, part := range candidate.Content.Parts {
// 			if part.Text != "" {
// 				llmResponse.Content += part.Text
// 				callback(part.Text)
// 			}
//
// 			if part.FunctionCall != nil {
// 				id := part.FunctionCall.ID
// 				if id == "" {
// 					id = fmt.Sprintf("call-%d", len(llmResponse.ToolCalls)+1)
// 				}
//
// 				llmResponse.ToolCalls = append(llmResponse.ToolCalls, types.ToolCall{
// 					ID:        id,
// 					Name:      part.FunctionCall.Name,
// 					Arguments: part.FunctionCall.Args,
// 				})
// 			}
// 		}
// 	}
//
// 	return llmResponse, nil
// }
//
// // func convertSchema(in *tools.Schema) *genai.Schema {
// // 	if in == nil {
// // 		return nil
// // 	}
// //
// // 	schema := &genai.Schema{
// // 		Type:        genai.Type(in.Type),
// // 		Description: in.Description,
// // 	}
// //
// // 	if in.Properties != nil {
// // 		schema.Properties = make(map[string]*genai.Schema, len(in.Properties))
// // 		for k, v := range in.Properties {
// // 			schema.Properties[k] = convertSchema(v)
// // 		}
// // 	}
// //
// // 	if in.Items != nil {
// // 		schema.Items = convertSchema(in.Items)
// // 	}
// //
// // 	return schema
// // }
