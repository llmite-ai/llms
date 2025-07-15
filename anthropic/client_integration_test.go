//go:build integration
// +build integration

package anthropic_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jpoz/llmite"
	"github.com/jpoz/llmite/anthropic"
	"github.com/jpoz/llmite/testutil"
)

type AnthropicTestSuite struct {
	suite.Suite
}

// func (suite *AnthropicTestSuite) SetupTest() {
// }

func (suite *AnthropicTestSuite) TestGenerateBasic() {
	suite.T().Parallel() // Enable parallel execution for this method
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithHttpLogging(),
	)

	msg := llmite.NewTextMessage(llmite.RoleUser, "What is the capital of France?")

	resp, err := client.Generate(ctx, []llmite.Message{msg})
	suite.NoError(err)
	suite.NotEmpty(resp.ID)
	suite.Require().NotEmpty(resp.Message.Parts)

	textPart := resp.Message.Parts[0]
	suite.IsType(llmite.TextPart{}, textPart)
}

func (suite *AnthropicTestSuite) TestGenerateSystemPrompt() {
	suite.T().Parallel() // Enable parallel execution for this method
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithHttpLogging(),
	)

	systemMsg := llmite.NewTextMessage(llmite.RoleSystem, "You only response in 'beep'  Example: beep beep beep.")
	userMsg := llmite.NewTextMessage(llmite.RoleUser, "Hello, how are you?")

	resp, err := client.Generate(ctx, []llmite.Message{systemMsg, userMsg})
	suite.Require().NoError(err)
	suite.NotEmpty(resp.ID)
	suite.Require().NotEmpty(resp.Message.Parts)

	textPart := resp.Message.Parts[0]
	suite.Require().IsType(llmite.TextPart{}, textPart)

	text, ok := textPart.(llmite.TextPart)
	suite.True(ok)
	suite.Contains(text.Text, "beep")
}

func (suite *AnthropicTestSuite) TestGenerateWithToolCalls() {
	suite.T().Parallel() // Enable parallel execution for this method
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithTools([]llmite.Tool{
			testutil.NewBoopTool(),
		}),
		anthropic.WithHttpLogging(),
	)

	systemMsg := llmite.NewTextMessage(llmite.RoleSystem, "You are a helpful assistant. That helps the user translate things.")
	userMsg := llmite.NewTextMessage(llmite.RoleUser, "My computer said `boop boop boop` when I turned it on. What does that mean?")

	resp, err := client.Generate(ctx, []llmite.Message{systemMsg, userMsg})
	suite.Require().NoError(err)
	suite.NotEmpty(resp.ID)
	suite.Require().NotEmpty(resp.Message.Parts)

	fmt.Printf("Response: %+v\n", resp)

	hasTextPart := false
	hasToolPart := false

	for _, part := range resp.Message.Parts {
		switch p := part.(type) {
		case llmite.TextPart:
			hasTextPart = true
		case llmite.ToolCallPart:
			hasToolPart = true

			var params testutil.BoopToolParams
			err := json.Unmarshal(p.Input, &params)
			suite.NoError(err)
		}
	}

	suite.True(hasTextPart, "response should have a text part")
	suite.True(hasToolPart, "response should have a tool call part")
}

func (suite *AnthropicTestSuite) TestGenerateStreamingWithToolCalls() {
	suite.T().Parallel() // Enable parallel execution for this method
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithTools([]llmite.Tool{
			testutil.NewBoopTool(),
		}),
		anthropic.WithHttpLogging(),
	)

	systemMsg := llmite.NewTextMessage(llmite.RoleSystem, "You are a helpful assistant. That helps the user translate things.")
	userMsg := llmite.NewTextMessage(llmite.RoleUser, "My computer said `boop boop boop` when I turned it on. What does that mean?")

	resp, err := client.GenerateStream(ctx, []llmite.Message{systemMsg, userMsg}, func(response *llmite.Response, err error) bool { return true })
	suite.Require().NoError(err)
	suite.NotEmpty(resp.ID)
	suite.Require().NotEmpty(resp.Message.Parts)

	fmt.Printf("Response: %+v\n", resp)

	hasTextPart := false
	hasToolPart := false

	for _, part := range resp.Message.Parts {
		switch p := part.(type) {
		case llmite.TextPart:
			hasTextPart = true
		case llmite.ToolCallPart:
			hasToolPart = true

			var params testutil.BoopToolParams
			err := json.Unmarshal(p.Input, &params)
			suite.NoError(err)
		}
	}

	suite.True(hasTextPart, "response should have a text part")
	suite.True(hasToolPart, "response should have a tool call part")
}

func TestAnthropicTestSuite(t *testing.T) {
	suite.Run(t, new(AnthropicTestSuite))
}
