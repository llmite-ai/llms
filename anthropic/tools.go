package anthropic

import (
	"net/http"

	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/invopop/jsonschema"
)

type BashTool struct {
}

func (b BashTool) Name() string { return "bash" }
func (b BashTool) Description() string {
	return "anthropic ran tool for executing bash commands"
}
func (b BashTool) Schema() *jsonschema.Schema { return nil }

type CodeExecutionTool struct {
}

func (c CodeExecutionTool) Name() string { return "code_execution" }

func (c CodeExecutionTool) Description() string { return "anthropic ran tool for executing code" }

func (c CodeExecutionTool) Schema() *jsonschema.Schema { return nil }

// CodeExecutionToolMiddleware is a middleware that sets the "anthropic-beta" header
func (c CodeExecutionTool) Middleware() option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		req.Header.Set("anthropic-beta", "code-execution-2025-05-22")
		return next(req)
	}
}
