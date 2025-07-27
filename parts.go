package llms // convertMessages converts the internal message format to the format

type TextPart struct {
	Text string `json:"text"`
}

func (TextPart) isPart() {}

type ToolCallPart struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input []byte `json:"arguments"`
}

func (ToolCallPart) isPart() {}

type ToolResultPart struct {
	ToolCallID string `json:"tool_call_id"`
	Name       string `json:"name"`
	Result     string `json:"result"`
	Error      error  `json:"error,omitempty"`
}

func (ToolResultPart) isPart() {}
