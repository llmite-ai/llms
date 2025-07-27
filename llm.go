package llms

import (
	"context"
)

type StreamFunc func(*Response, error) bool

type LLM interface {
	Generate(ctx context.Context, messages []Message) (*Response, error)
	GenerateStream(ctx context.Context, messages []Message, fn StreamFunc) (*Response, error)
}

type Part interface {
	isPart()
}

type Response struct {
	ID      string  `json:"id"`
	Message Message `json:"message"`

	Provider string
	Raw      any
}
