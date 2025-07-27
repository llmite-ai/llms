package llms

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role  Role   `json:"role"`
	Parts []Part `json:"parts"`
}

func NewTextMessage(role Role, text string) Message {
	return Message{
		Role:  role,
		Parts: []Part{TextPart{Text: text}},
	}
}
