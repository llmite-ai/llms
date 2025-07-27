package anthropic

import "encoding/json"

type ServerToolUsePart struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

func (ServerToolUsePart) IsPart() {}

type CodeExecutionToolResult struct {
	ToolUseID string              `json:"tool_use_id"`
	Content   CodeExecutionResult `json:"content"` // This contains the result of the code execution
}

func (CodeExecutionToolResult) IsPart() {}

type CodeExecutionResult struct {
	Stdout     string          `json:"stdout"`
	Stderr     string          `json:"stderr"`
	ReturnCode int             `json:"return_code"`
	Content    json.RawMessage `json:"content"` // This can be used for additional content if needed
}
