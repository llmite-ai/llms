package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

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

	baseURL string
	client  *http.Client
	token   string
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

// WithBaseURL allows you to set the base URL on the client. This is useful if you want to
// set a custom base URL instead of using the default "https://api.anthropic.com".
func WithBaseURL(url string) Modifer {
	return func(a *Client) {
		a.baseURL = url
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
		baseURL:   "https://api.anthropic.com",
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
	request, err := a.buildRequest(messages, false)
	if err != nil {
		return nil, err
	}

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
	request, err := a.buildRequest(messages, true)
	if err != nil {
		return nil, err
	}

	response, err := a.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("anthropic: request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("anthropic: unexpected status code: %d", response.StatusCode)
	}

	// Parse the streaming response
	finalResponse, err := a.parseSSEStream(ctx, response.Body, fn)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to parse stream: %w", err)
	}

	return finalResponse, nil
}

func (a *Client) buildRequest(messages []llmite.Message, streaming bool) (*http.Request, error) {
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

	if streaming {
		requestBody.Stream = &streaming
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

	url, err := url.JoinPath(a.baseURL, "/v1/messages")
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to join URL path: %w (%s,%s)", err, a.baseURL, "/v1/messages")
	}

	request, err := http.NewRequest("POST", url, bodyBuffer)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("anthropic-version", Version)
	request.Header.Set("x-api-key", a.token)

	return request, nil
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
				var input map[string]any

				err := json.Unmarshal(p.Input, &input)
				if err != nil {
					return nil, nil, fmt.Errorf("anthropic: failed to unmarshal tool call input JSON for tool '%s': %w", p.Name, err)
				}

				block := RequestToolUseBlock{
					ID:    p.ID,
					Name:  p.Name,
					Input: input,
				}
				inputMsg.Content = append(inputMsg.Content, block)
			case llmite.ToolResultPart:
				block := RequestToolResultBlock{
					ToolUseID: p.ToolCallID,
					Content:   p.Result,
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

func (a *Client) parseSSEStream(ctx context.Context, body io.Reader, fn llmite.StreamFunc) (*llmite.Response, error) {
	scanner := bufio.NewScanner(body)

	var anthropicResponse *CreateMessageResponse
	var currentEvent StreamEvent
	var streamingParts []llmite.Part
	var currentTextBuilder strings.Builder
	var currentToolCall *llmite.ToolCallPart
	var responseID string

	for scanner.Scan() {
		line := scanner.Text()

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Parse SSE format
		if strings.HasPrefix(line, "event: ") {
			currentEvent.Event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			currentEvent.Data = strings.TrimPrefix(line, "data: ")
		} else if line == "" {
			// Empty line indicates end of event, process it
			if currentEvent.Event != "" && currentEvent.Data != "" {
				shouldContinue, err := a.processStreamEvent(
					currentEvent,
					fn,
					&anthropicResponse,
					&streamingParts,
					&currentTextBuilder,
					&currentToolCall,
					&responseID,
				)
				if err != nil {
					return nil, err
				}
				if !shouldContinue {
					break
				}
			}
			// Reset for next event
			currentEvent = StreamEvent{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	if anthropicResponse == nil {
		return nil, fmt.Errorf("no message received in stream")
	}

	// Build final response
	finalResponse := &llmite.Response{
		ID: responseID,
		Message: llmite.Message{
			Role:  llmite.RoleAssistant,
			Parts: streamingParts,
		},
		Provider: "anthropic",
		Raw:      anthropicResponse,
	}

	return finalResponse, nil
}

func (a *Client) processStreamEvent(
	event StreamEvent,
	fn llmite.StreamFunc,
	anthropicResponse **CreateMessageResponse,
	streamingParts *[]llmite.Part,
	currentTextBuilder *strings.Builder,
	currentToolCall **llmite.ToolCallPart,
	responseID *string,
) (bool, error) {
	switch event.Event {
	case "message_start":
		var msgStart MessageStartEvent
		if err := json.Unmarshal([]byte(event.Data), &msgStart); err != nil {
			return false, fmt.Errorf("failed to unmarshal message_start: %w", err)
		}
		*anthropicResponse = &msgStart.Message
		*responseID = msgStart.Message.ID

	case "content_block_start":
		var blockStart ContentBlockStartEvent
		if err := json.Unmarshal([]byte(event.Data), &blockStart); err != nil {
			return false, fmt.Errorf("failed to unmarshal content_block_start: %w", err)
		}

		// Initialize based on content block type
		switch blockStart.ContentBlock.Type {
		case "text":
			currentTextBuilder.Reset()
		case "tool_use":
			// Parse the tool use block to get ID and name
			*currentToolCall = &llmite.ToolCallPart{
				ID:    blockStart.ContentBlock.ID,
				Name:  blockStart.ContentBlock.Name,
				Input: json.RawMessage("{}"), // Will be built up from deltas
			}
		}

	case "content_block_delta":
		var blockDelta ContentBlockDeltaEvent
		if err := json.Unmarshal([]byte(event.Data), &blockDelta); err != nil {
			return false, fmt.Errorf("failed to unmarshal content_block_delta: %w", err)
		}

		switch blockDelta.Delta.Type {
		case "text_delta":
			currentTextBuilder.WriteString(blockDelta.Delta.Text)

			// Create a streaming response with the current state
			if fn != nil {
				currentParts := make([]llmite.Part, len(*streamingParts))
				copy(currentParts, *streamingParts)

				// Add the current text being built
				if currentTextBuilder.Len() > 0 {
					currentParts = append(currentParts, llmite.TextPart{
						Text: currentTextBuilder.String(),
					})
				}

				streamResponse := &llmite.Response{
					ID: *responseID,
					Message: llmite.Message{
						Role:  llmite.RoleAssistant,
						Parts: currentParts,
					},
					Provider: "anthropic",
					Raw:      *anthropicResponse,
				}

				shouldContinue := fn(streamResponse, nil)
				if !shouldContinue {
					return false, nil
				}
			}

		case "input_json_delta":
			// Build up the tool input JSON
			if *currentToolCall != nil {
				// Append the partial JSON to build the complete input
				currentInput := string((*currentToolCall).Input)
				if currentInput == "{}" {
					currentInput = ""
				}
				currentInput += blockDelta.Delta.PartialJSON
				(*currentToolCall).Input = json.RawMessage(currentInput)
			}

		case "thinking_delta":
			// For now, we'll ignore thinking deltas

		case "signature_delta":
			// For now, we'll ignore signature deltas
		}

	case "content_block_stop":
		var blockStop ContentBlockStopEvent
		if err := json.Unmarshal([]byte(event.Data), &blockStop); err != nil {
			return false, fmt.Errorf("failed to unmarshal content_block_stop: %w", err)
		}

		// Finalize the current content block
		if currentTextBuilder.Len() > 0 {
			*streamingParts = append(*streamingParts, llmite.TextPart{
				Text: currentTextBuilder.String(),
			})
			currentTextBuilder.Reset()
		}

		if *currentToolCall != nil {
			*streamingParts = append(*streamingParts, **currentToolCall)
			*currentToolCall = nil
		}

	case "message_delta":
		var msgDelta MessageDeltaEvent
		if err := json.Unmarshal([]byte(event.Data), &msgDelta); err != nil {
			return false, fmt.Errorf("failed to unmarshal message_delta: %w", err)
		}

		if *anthropicResponse != nil {
			if msgDelta.Delta.StopReason != nil {
				(*anthropicResponse).StopReason = msgDelta.Delta.StopReason
			}
			if msgDelta.Delta.StopSequence != nil {
				(*anthropicResponse).StopSequence = msgDelta.Delta.StopSequence
			}
			if msgDelta.Usage != nil {
				(*anthropicResponse).Usage = *msgDelta.Usage
			}
		}

	case "message_stop":
		// Stream is complete - send final response
		if fn != nil {
			finalResponse := &llmite.Response{
				ID: *responseID,
				Message: llmite.Message{
					Role:  llmite.RoleAssistant,
					Parts: *streamingParts,
				},
				Provider: "anthropic",
				Raw:      *anthropicResponse,
			}
			fn(finalResponse, nil)
		}

	case "ping":
		// Just a keep-alive, ignore

	case "error":
		var errEvent ErrorEvent
		if err := json.Unmarshal([]byte(event.Data), &errEvent); err != nil {
			return false, fmt.Errorf("failed to unmarshal error event: %w", err)
		}

		if fn != nil {
			fn(nil, fmt.Errorf("stream error: %s", errEvent.Error.Message))
		}
		return false, fmt.Errorf("stream error: %s", errEvent.Error.Message)

	default:
		return false, fmt.Errorf("unknown stream event type: %s", event.Event)
	}

	return true, nil
}
