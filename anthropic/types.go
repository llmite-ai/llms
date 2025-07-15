package anthropic

import (
	"encoding/json"
	"time"
)

// ============================================================================
// CONTENT BLOCK INTERFACE
// ============================================================================

// ContentBlock represents any type of content block
type ContentBlock interface {
	GetType() string
}

// RequestContentBlock represents content blocks that can be sent in requests
type RequestContentBlock interface {
	ContentBlock
}

// ResponseContentBlock represents content blocks that can be received in responses
type ResponseContentBlock interface {
	ContentBlock
}

// ============================================================================
// REQUEST TYPES
// ============================================================================

// CreateMessageRequest represents the request payload for creating a message
type CreateMessageRequest struct {
	Model         string                `json:"model"`      // Required
	Messages      []InputMessage        `json:"messages"`   // Required
	MaxTokens     int                   `json:"max_tokens"` // Required
	Container     *string               `json:"container,omitempty"`
	MCPServers    []MCPServerDefinition `json:"mcp_servers,omitempty"`
	Metadata      *Metadata             `json:"metadata,omitempty"`
	ServiceTier   *string               `json:"service_tier,omitempty"` // "auto", "standard_only"
	StopSequences []string              `json:"stop_sequences,omitempty"`
	Stream        *bool                 `json:"stream,omitempty"`
	System        SystemPrompt          `json:"system,omitempty"`
	Temperature   *float64              `json:"temperature,omitempty"` // 0.0 - 1.0
	Thinking      *ThinkingConfig       `json:"thinking,omitempty"`
	ToolChoice    *ToolChoice           `json:"tool_choice,omitempty"`
	Tools         []Tool                `json:"tools,omitempty"`
	TopK          *int                  `json:"top_k,omitempty"`
	TopP          *float64              `json:"top_p,omitempty"` // 0.0 - 1.0
}

// InputMessage represents a message in the conversation
type InputMessage struct {
	Role    string         `json:"role"` // "user" or "assistant"
	Content []ContentBlock `json:"content"`
}

// Metadata contains request metadata
type Metadata struct {
	UserID *string `json:"user_id,omitempty"`
}

// SystemPrompt can be either a string or array of text blocks
type SystemPrompt []RequestTextBlock

// ThinkingType represents the type of thinking configuration
type ThinkingType string

const ThinkingTypeEnabled = "enabled"
const ThinkingTypeDisabled = "disabled"

// ThinkingConfig controls Claude's extended thinking
type ThinkingConfig struct {
	Type         ThinkingType `json:"type"`                    // "enabled" or "disabled"
	BudgetTokens *int         `json:"budget_tokens,omitempty"` // Required if enabled, min 1024
}

// MCPServerDefinition defines an MCP server
type MCPServerDefinition struct {
	Type               URLType                     `json:"type"` // "url"
	Name               string                      `json:"name"`
	URL                string                      `json:"url"`
	AuthorizationToken *string                     `json:"authorization_token,omitempty"`
	ToolConfiguration  *MCPServerToolConfiguration `json:"tool_configuration,omitempty"`
}

// MCPServerToolConfiguration configures MCP server tools
type MCPServerToolConfiguration struct {
	Enabled      *bool    `json:"enabled,omitempty"`
	AllowedTools []string `json:"allowed_tools,omitempty"`
}

// ============================================================================
// REQUEST CONTENT BLOCKS
// ============================================================================

// RequestTextBlock represents text content
type RequestTextBlock struct {
	Type         TextType      `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
	Citations    []Citation    `json:"citations,omitempty"`
}

func (r RequestTextBlock) GetType() string { return "text" }

// RequestImageBlock represents image content
type RequestImageBlock struct {
	Type         ImageType     `json:"type"` // "image"
	Source       ImageSource   `json:"source"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

func (r RequestImageBlock) GetType() string { return "image" }

// RequestDocumentBlock represents document content
type RequestDocumentBlock struct {
	Type         DocumentType     `json:"type"` // "document"
	Source       DocumentSource   `json:"source"`
	Title        *string          `json:"title,omitempty"`
	Context      *string          `json:"context,omitempty"`
	Citations    *CitationsConfig `json:"citations,omitempty"`
	CacheControl *CacheControl    `json:"cache_control,omitempty"`
}

func (r RequestDocumentBlock) GetType() string { return "document" }

// RequestToolUseBlock represents tool usage
type RequestToolUseBlock struct {
	Type         ToolUseType    `json:"type"` // "tool_use"
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Input        map[string]any `json:"input"`
	CacheControl *CacheControl  `json:"cache_control,omitempty"`
}

func (r RequestToolUseBlock) GetType() string { return "tool_use" }

// RequestToolResultBlock represents tool results
type RequestToolResultBlock struct {
	Type         ToolResultType `json:"type"` // "tool_result"
	ToolUseID    string         `json:"tool_use_id"`
	Content      any            `json:"content,omitempty"` // string or []ContentBlock
	IsError      *bool          `json:"is_error,omitempty"`
	CacheControl *CacheControl  `json:"cache_control,omitempty"`
}

func (r RequestToolResultBlock) GetType() string { return "tool_result" }

// RequestThinkingBlock represents thinking content in requests
type RequestThinkingBlock struct {
	Type      ThinkingContentType `json:"type"` // "thinking"
	Thinking  string              `json:"thinking"`
	Signature string              `json:"signature"`
}

func (r RequestThinkingBlock) GetType() string { return "thinking" }

// RequestRedactedThinkingBlock represents redacted thinking content
type RequestRedactedThinkingBlock struct {
	Type string `json:"type"` // "redacted_thinking"
	Data string `json:"data"`
}

func (r RequestRedactedThinkingBlock) GetType() string { return "redacted_thinking" }

// RequestContainerUploadBlock represents a file upload to container
type RequestContainerUploadBlock struct {
	Type         ContainerUploadType `json:"type"` // "container_upload"
	FileID       string              `json:"file_id"`
	CacheControl *CacheControl       `json:"cache_control,omitempty"`
}

func (r RequestContainerUploadBlock) GetType() string { return "container_upload" }

// RequestServerToolUseBlock represents server tool usage
type RequestServerToolUseBlock struct {
	Type         ServerToolUseType `json:"type"` // "server_tool_use"
	ID           string            `json:"id"`
	Name         string            `json:"name"` // "web_search", "code_execution"
	Input        map[string]any    `json:"input"`
	CacheControl *CacheControl     `json:"cache_control,omitempty"`
}

func (r RequestServerToolUseBlock) GetType() string { return "server_tool_use" }

// RequestMCPToolUseBlock represents MCP tool usage
type RequestMCPToolUseBlock struct {
	Type         MCPToolUseType `json:"type"` // "mcp_tool_use"
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	ServerName   string         `json:"server_name"`
	Input        map[string]any `json:"input"`
	CacheControl *CacheControl  `json:"cache_control,omitempty"`
}

func (r RequestMCPToolUseBlock) GetType() string { return "mcp_tool_use" }

// RequestMCPToolResultBlock represents MCP tool results
type RequestMCPToolResultBlock struct {
	Type         MCPToolResultType `json:"type"` // "mcp_tool_result"
	ToolUseID    string            `json:"tool_use_id"`
	Content      any               `json:"content,omitempty"` // string or []RequestTextBlock
	IsError      *bool             `json:"is_error,omitempty"`
	CacheControl *CacheControl     `json:"cache_control,omitempty"`
}

func (r RequestMCPToolResultBlock) GetType() string { return "mcp_tool_result" }

// RequestWebSearchToolResultBlock represents web search tool results
type RequestWebSearchToolResultBlock struct {
	Type         WebSearchToolResultType `json:"type"` // "web_search_tool_result"
	ToolUseID    string                  `json:"tool_use_id"`
	Content      any                     `json:"content"` // []RequestWebSearchResultBlock or error
	CacheControl *CacheControl           `json:"cache_control,omitempty"`
}

func (r RequestWebSearchToolResultBlock) GetType() string { return "web_search_tool_result" }

// RequestWebSearchResultBlock represents a web search result in requests
type RequestWebSearchResultBlock struct {
	Type             WebSearchResultType `json:"type"` // "web_search_result"
	Title            string              `json:"title"`
	URL              string              `json:"url"`
	EncryptedContent string              `json:"encrypted_content"`
	PageAge          *string             `json:"page_age,omitempty"`
}

func (r RequestWebSearchResultBlock) GetType() string { return "web_search_result" }

// RequestCodeExecutionToolResultBlock represents code execution tool results
type RequestCodeExecutionToolResultBlock struct {
	Type         CodeExecutionToolResultType `json:"type"` // "code_execution_tool_result"
	ToolUseID    string                      `json:"tool_use_id"`
	Content      any                         `json:"content"` // RequestCodeExecutionResultBlock or error
	CacheControl *CacheControl               `json:"cache_control,omitempty"`
}

func (r RequestCodeExecutionToolResultBlock) GetType() string { return "code_execution_tool_result" }

// RequestCodeExecutionResultBlock represents code execution result details
type RequestCodeExecutionResultBlock struct {
	Type       CodeExecutionResultType           `json:"type"` // "code_execution_result"
	Stdout     string                            `json:"stdout"`
	Stderr     string                            `json:"stderr"`
	ReturnCode int                               `json:"return_code"`
	Content    []RequestCodeExecutionOutputBlock `json:"content"`
}

func (r RequestCodeExecutionResultBlock) GetType() string { return "code_execution_result" }

// RequestCodeExecutionOutputBlock represents code execution output
type RequestCodeExecutionOutputBlock struct {
	Type   CodeExecutionOutputType `json:"type"` // "code_execution_output"
	FileID string                  `json:"file_id"`
}

func (r RequestCodeExecutionOutputBlock) GetType() string { return "code_execution_output" }

type ImageSourceType string

const ImageSourceTypeBase64 = "base64"
const ImageSourceTypeURL = "url"
const ImageSourceTypeFile = "file"

// ImageSource defines image source
type ImageSource struct {
	Type      ImageSourceType `json:"type"`                 // "base64", "url", or "file"
	Data      string          `json:"data,omitempty"`       // base64 data
	MediaType string          `json:"media_type,omitempty"` // "image/jpeg", "image/png", etc.
	URL       string          `json:"url,omitempty"`        // for URL type
	FileID    string          `json:"file_id,omitempty"`    // for file type
}

// DocumentSource defines document source
type DocumentSource struct {
	Type      string `json:"type"` // "base64", "text", "content", "url", or "file"
	Data      string `json:"data,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Content   any    `json:"content,omitempty"` // string or []ContentBlock
	URL       string `json:"url,omitempty"`
	FileID    string `json:"file_id,omitempty"`
}

// CacheControl for caching configuration
type CacheControl struct {
	Type string `json:"type"`          // "ephemeral"
	TTL  string `json:"ttl,omitempty"` // "5m" or "1h"
}

// CitationsConfig controls citation generation
type CitationsConfig struct {
	Enabled bool `json:"enabled"`
}

// Citation represents a citation
type Citation struct {
	Type          string  `json:"type"` // "char_location", "page_location", etc.
	CitedText     string  `json:"cited_text"`
	DocumentIndex int     `json:"document_index"`
	DocumentTitle *string `json:"document_title"`
	// Additional fields based on citation type
	StartCharIndex  *int `json:"start_char_index,omitempty"`
	EndCharIndex    *int `json:"end_char_index,omitempty"`
	StartPageNumber *int `json:"start_page_number,omitempty"`
	EndPageNumber   *int `json:"end_page_number,omitempty"`
	StartBlockIndex *int `json:"start_block_index,omitempty"`
	EndBlockIndex   *int `json:"end_block_index,omitempty"`
}

// ============================================================================
// TOOLS
// ============================================================================

// Tool represents a tool definition
type Tool struct {
	Type         *string       `json:"type,omitempty"` // "custom" or null
	Name         string        `json:"name"`
	Description  *string       `json:"description,omitempty"`
	InputSchema  InputSchema   `json:"input_schema"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// InputSchema defines the JSON schema for tool input
type InputSchema struct {
	Type                 ObjectType     `json:"type"` // "object"
	Properties           map[string]any `json:"properties,omitempty"`
	Required             []string       `json:"required,omitempty"`
	AdditionalProperties any            `json:"additionalProperties,omitempty"`
}

// ToolChoice controls tool selection
type ToolChoice struct {
	Type                   string `json:"type"`           // "auto", "any", "tool", "none"
	Name                   string `json:"name,omitempty"` // required if type is "tool"
	DisableParallelToolUse *bool  `json:"disable_parallel_tool_use,omitempty"`
}

// ============================================================================
// RESPONSE TYPES
// ============================================================================

// CreateMessageResponse represents the API response
type CreateMessageResponse struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"` // "message"
	Role         string            `json:"role"` // "assistant"
	Content      []ResponseContent `json:"content"`
	Model        string            `json:"model"`
	StopReason   *string           `json:"stop_reason"` // "end_turn", "max_tokens", etc.
	StopSequence *string           `json:"stop_sequence"`
	Usage        Usage             `json:"usage"`
	Container    *Container        `json:"container"`
}

// ResponseContent represents different types of response content
type ResponseContent interface {
	GetType() string
}

func (r *CreateMessageResponse) UnmarshalJSON(data []byte) error {
	type Alias CreateMessageResponse
	aux := &struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	contents, err := UnmarshalResponseContent(aux.Content)
	if err != nil {
		return err
	}
	r.Content = contents

	return nil
}

// ============================================================================
// RESPONSE CONTENT BLOCKS
// ============================================================================

// ResponseTextBlock represents text content in response
type ResponseTextBlock struct {
	Type      string     `json:"type"` // "text"
	Text      string     `json:"text"`
	Citations []Citation `json:"citations"`
}

func (r ResponseTextBlock) GetType() string { return "text" }

// ResponseToolUseBlock represents tool usage in response
type ResponseToolUseBlock struct {
	Type  string          `json:"type"` // "tool_use"
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (r ResponseToolUseBlock) GetType() string { return "tool_use" }

// ResponseThinkingBlock represents thinking content
type ResponseThinkingBlock struct {
	Type      string `json:"type"` // "thinking"
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

func (r ResponseThinkingBlock) GetType() string { return "thinking" }

// ResponseRedactedThinkingBlock represents redacted thinking content in response
type ResponseRedactedThinkingBlock struct {
	Type string `json:"type"` // "redacted_thinking"
	Data string `json:"data"`
}

func (r ResponseRedactedThinkingBlock) GetType() string { return "redacted_thinking" }

// ResponseServerToolUseBlock represents server tool usage in response
type ResponseServerToolUseBlock struct {
	Type  string         `json:"type"` // "server_tool_use"
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

func (r ResponseServerToolUseBlock) GetType() string { return "server_tool_use" }

// ResponseMCPToolUseBlock represents MCP tool usage in response
type ResponseMCPToolUseBlock struct {
	Type       string         `json:"type"` // "mcp_tool_use"
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	ServerName string         `json:"server_name"`
	Input      map[string]any `json:"input"`
}

func (r ResponseMCPToolUseBlock) GetType() string { return "mcp_tool_use" }

// ResponseMCPToolResultBlock represents MCP tool results in response
type ResponseMCPToolResultBlock struct {
	Type      string `json:"type"` // "mcp_tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"` // string or []ResponseTextBlock
	IsError   bool   `json:"is_error"`
}

func (r ResponseMCPToolResultBlock) GetType() string { return "mcp_tool_result" }

// ResponseContainerUploadBlock represents container upload response
type ResponseContainerUploadBlock struct {
	Type   string `json:"type"` // "container_upload"
	FileID string `json:"file_id"`
}

func (r ResponseContainerUploadBlock) GetType() string { return "container_upload" }

// ResponseWebSearchToolResultBlock represents web search results
type ResponseWebSearchToolResultBlock struct {
	Type      string `json:"type"` // "web_search_tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"` // []ResponseWebSearchResultBlock or error
}

func (r ResponseWebSearchToolResultBlock) GetType() string { return "web_search_tool_result" }

// ResponseWebSearchResultBlock represents a single web search result
type ResponseWebSearchResultBlock struct {
	Type             string  `json:"type"` // "web_search_result"
	Title            string  `json:"title"`
	URL              string  `json:"url"`
	EncryptedContent string  `json:"encrypted_content"`
	PageAge          *string `json:"page_age"`
}

func (r ResponseWebSearchResultBlock) GetType() string { return "web_search_result" }

// ResponseCodeExecutionToolResultBlock represents code execution results
type ResponseCodeExecutionToolResultBlock struct {
	Type      string `json:"type"` // "code_execution_tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"` // ResponseCodeExecutionResultBlock or error
}

func (r ResponseCodeExecutionToolResultBlock) GetType() string { return "code_execution_tool_result" }

// ResponseCodeExecutionResultBlock represents code execution result details
type ResponseCodeExecutionResultBlock struct {
	Type       string                             `json:"type"` // "code_execution_result"
	Stdout     string                             `json:"stdout"`
	Stderr     string                             `json:"stderr"`
	ReturnCode int                                `json:"return_code"`
	Content    []ResponseCodeExecutionOutputBlock `json:"content"`
}

func (r ResponseCodeExecutionResultBlock) GetType() string { return "code_execution_result" }

// ResponseCodeExecutionOutputBlock represents code execution output
type ResponseCodeExecutionOutputBlock struct {
	Type   string `json:"type"` // "code_execution_output"
	FileID string `json:"file_id"`
}

func (r ResponseCodeExecutionOutputBlock) GetType() string { return "code_execution_output" }

// Usage represents token usage information
type Usage struct {
	InputTokens              int              `json:"input_tokens"`
	OutputTokens             int              `json:"output_tokens"`
	CacheCreationInputTokens *int             `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int             `json:"cache_read_input_tokens"`
	CacheCreation            *CacheCreation   `json:"cache_creation"`
	ServerToolUse            *ServerToolUsage `json:"server_tool_use"`
	ServiceTier              *string          `json:"service_tier"`
}

// CacheCreation provides cache creation breakdown
type CacheCreation struct {
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
}

// ServerToolUsage tracks server tool usage
type ServerToolUsage struct {
	WebSearchRequests int `json:"web_search_requests"`
}

// Container represents container information
type Container struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ============================================================================
// ERROR TYPES
// ============================================================================

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Type  string `json:"type"` // "error"
	Error Error  `json:"error"`
}

// Error represents an API error
type Error struct {
	Type    string `json:"type"` // "invalid_request_error", "authentication_error", etc.
	Message string `json:"message"`
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// AddTextBlock adds a text block to multi-content
func AddTextBlock(text string) RequestTextBlock {
	return RequestTextBlock{
		Text: text,
	}
}

// AddImageBlock adds an image block with base64 data
func AddImageBlock(data, mediaType string) RequestImageBlock {
	return RequestImageBlock{
		Source: ImageSource{
			Type:      ImageSourceTypeBase64,
			Data:      data,
			MediaType: mediaType,
		},
	}
}

// UnmarshalResponseContent properly unmarshals response content based on type
func UnmarshalResponseContent(data []byte) ([]ResponseContent, error) {
	var rawContents []json.RawMessage
	if err := json.Unmarshal(data, &rawContents); err != nil {
		return nil, err
	}

	var contents []ResponseContent
	for _, raw := range rawContents {
		var typeChecker struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeChecker); err != nil {
			continue
		}

		switch typeChecker.Type {
		case "text":
			var block ResponseTextBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "tool_use":
			var block ResponseToolUseBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "thinking":
			var block ResponseThinkingBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "redacted_thinking":
			var block ResponseRedactedThinkingBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "server_tool_use":
			var block ResponseServerToolUseBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "mcp_tool_use":
			var block ResponseMCPToolUseBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "mcp_tool_result":
			var block ResponseMCPToolResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "container_upload":
			var block ResponseContainerUploadBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "web_search_tool_result":
			var block ResponseWebSearchToolResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "web_search_result":
			var block ResponseWebSearchResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "code_execution_tool_result":
			var block ResponseCodeExecutionToolResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "code_execution_result":
			var block ResponseCodeExecutionResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "code_execution_output":
			var block ResponseCodeExecutionOutputBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		}
	}

	return contents, nil
}

// UnmarshalRequestContentBlocks properly unmarshals request content blocks
func UnmarshalRequestContentBlocks(data []byte) ([]RequestContentBlock, error) {
	var rawContents []json.RawMessage
	if err := json.Unmarshal(data, &rawContents); err != nil {
		return nil, err
	}

	var contents []RequestContentBlock
	for _, raw := range rawContents {
		var typeChecker struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeChecker); err != nil {
			continue
		}

		switch typeChecker.Type {
		case "text":
			var block RequestTextBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "image":
			var block RequestImageBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "document":
			var block RequestDocumentBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "tool_use":
			var block RequestToolUseBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "tool_result":
			var block RequestToolResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "thinking":
			var block RequestThinkingBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "redacted_thinking":
			var block RequestRedactedThinkingBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "container_upload":
			var block RequestContainerUploadBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "server_tool_use":
			var block RequestServerToolUseBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "mcp_tool_use":
			var block RequestMCPToolUseBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "mcp_tool_result":
			var block RequestMCPToolResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "web_search_tool_result":
			var block RequestWebSearchToolResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "web_search_result":
			var block RequestWebSearchResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "code_execution_tool_result":
			var block RequestCodeExecutionToolResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "code_execution_result":
			var block RequestCodeExecutionResultBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		case "code_execution_output":
			var block RequestCodeExecutionOutputBlock
			if err := json.Unmarshal(raw, &block); err == nil {
				contents = append(contents, block)
			}
		}
	}

	return contents, nil
}

// ============================================================================
// STREAMING EVENT TYPES
// ============================================================================

// StreamEvent represents a server-sent event
type StreamEvent struct {
	Event string `json:"-"`
	Data  string `json:"-"`
}

// Streaming event types
type MessageStartEvent struct {
	Type    string                `json:"type"`
	Message CreateMessageResponse `json:"message"`
}

type ContentBlockStartEvent struct {
	Type         string                `json:"type"`
	Index        int                   `json:"index"`
	ContentBlock StreamingContentBlock `json:"content_block"`
}

type ContentBlockDeltaEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta Delta  `json:"delta"`
}

type ContentBlockStopEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type MessageDeltaEvent struct {
	Type  string       `json:"type"`
	Delta MessageDelta `json:"delta"`
	Usage *Usage       `json:"usage,omitempty"`
}

type MessageStopEvent struct {
	Type string `json:"type"`
}

type PingEvent struct {
	Type string `json:"type"`
}

type ErrorEvent struct {
	Type  string `json:"type"`
	Error Error  `json:"error"`
}

// Delta types for content block deltas
type Delta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`         // for text_delta
	PartialJSON string `json:"partial_json,omitempty"` // for input_json_delta
	Thinking    string `json:"thinking,omitempty"`     // for thinking_delta
	Signature   string `json:"signature,omitempty"`    // for signature_delta
}

// MessageDelta for message-level changes
type MessageDelta struct {
	StopReason   *string `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

// StreamingContentBlock for content block start events
type StreamingContentBlock struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name,omitempty"`
	Input    map[string]any `json:"input,omitempty"`
	Thinking string         `json:"thinking,omitempty"`
}

func (s StreamingContentBlock) GetType() string {
	return s.Type
}

// ============================================================================
// CONTENT BLOCK TYPES (for JSON marshalling)
// ============================================================================

type TextType struct{}

func (t TextType) MarshalJSON() ([]byte, error) { return []byte(`"text"`), nil }

type URLType struct{}

func (t URLType) MarshalJSON() ([]byte, error) { return []byte(`"url"`), nil }

type ImageType struct{}

func (t ImageType) MarshalJSON() ([]byte, error) { return []byte(`"image"`), nil }

type DocumentType struct{}

func (t DocumentType) MarshalJSON() ([]byte, error) { return []byte(`"document"`), nil }

type ObjectType struct{}

func (t ObjectType) MarshalJSON() ([]byte, error) { return []byte(`"object"`), nil }

type ToolUseType struct{}

func (t ToolUseType) MarshalJSON() ([]byte, error) { return []byte(`"tool_use"`), nil }

type ToolResultType struct{}

func (t ToolResultType) MarshalJSON() ([]byte, error) { return []byte(`"tool_result"`), nil }

type ThinkingContentType struct{}

func (t ThinkingContentType) MarshalJSON() ([]byte, error) { return []byte(`"thinking"`), nil }

type RedactedThinkingType struct{}

func (t RedactedThinkingType) MarshalJSON() ([]byte, error) {
	return []byte(`"redacted_thinking"`), nil
}

type ContainerUploadType struct{}

func (t ContainerUploadType) MarshalJSON() ([]byte, error) { return []byte(`"container_upload"`), nil }

type ServerToolUseType struct{}

func (t ServerToolUseType) MarshalJSON() ([]byte, error) { return []byte(`"server_tool_use"`), nil }

type MCPToolUseType struct{}

func (t MCPToolUseType) MarshalJSON() ([]byte, error) { return []byte(`"mcp_tool_use"`), nil }

type MCPToolResultType struct{}

func (t MCPToolResultType) MarshalJSON() ([]byte, error) { return []byte(`"mcp_tool_result"`), nil }

type WebSearchToolResultType struct{}

func (t WebSearchToolResultType) MarshalJSON() ([]byte, error) {
	return []byte(`"web_search_tool_result"`), nil
}

type WebSearchResultType struct{}

func (t WebSearchResultType) MarshalJSON() ([]byte, error) { return []byte(`"web_search_result"`), nil }

type CodeExecutionToolResultType struct{}

func (t CodeExecutionToolResultType) MarshalJSON() ([]byte, error) {
	return []byte(`"code_execution_tool_result"`), nil
}

type CodeExecutionResultType struct{}

func (t CodeExecutionResultType) MarshalJSON() ([]byte, error) {
	return []byte(`"code_execution_result"`), nil
}

type CodeExecutionOutputType struct{}

func (t CodeExecutionOutputType) MarshalJSON() ([]byte, error) {
	return []byte(`"code_execution_output"`), nil
}
