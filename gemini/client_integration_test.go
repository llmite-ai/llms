//go:build integration
// +build integration

package gemini_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/llmite-ai/llms"
	"github.com/llmite-ai/llms/gemini"
	"github.com/llmite-ai/llms/testutil"
)

type GeminiTestSuite struct {
	suite.Suite
}

func (suite *GeminiTestSuite) TestGenerateStreamBasic() {
	suite.T().Parallel()
	ctx := context.Background()
	client, err := gemini.New(
		gemini.WithModel("gemini-2.5-flash"),
		gemini.WithHttpLogging(),
	)
	suite.Require().NoError(err)

	msg := llms.NewTextMessage(llms.RoleUser, "What is the meaning of life?")

	resp, err := client.GenerateStream(ctx, []llms.Message{msg}, func(response *llms.Response, err error) bool {
		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			return false
		}
		fmt.Printf("---\n%v\n---\n", response)
		return true
	})
	suite.NoError(err)
	suite.Require().NotEmpty(resp.Message.Parts)

	textPart := resp.Message.Parts[0]
	suite.IsType(llms.TextPart{}, textPart)
}

func (suite *GeminiTestSuite) TestGenerateStreamSystemPrompt() {
	suite.T().Parallel()
	ctx := context.Background()
	client, err := gemini.New(
		gemini.WithSystemInstruction("You only response in 'beep'  Example: beep beep beep."),
		gemini.WithHttpLogging(),
	)
	suite.Require().NoError(err)

	userMsg := llms.NewTextMessage(llms.RoleUser, "What is your favorite color?")

	resp, err := client.GenerateStream(ctx, []llms.Message{userMsg}, func(response *llms.Response, err error) bool {
		fmt.Println("Stream chunk:", response)
		return true
	})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(resp.Message.Parts)

	textPart := resp.Message.Parts[0]
	suite.Require().IsType(llms.TextPart{}, textPart)

	text, ok := textPart.(llms.TextPart)
	suite.True(ok)
	suite.Contains(text.Text, "beep")
}

func (suite *GeminiTestSuite) TestGenerateStreamWithToolCalls() {
	suite.T().Parallel()
	ctx := context.Background()
	client, err := gemini.New(
		gemini.WithTools([]llms.Tool{
			testutil.NewBoopTool(),
		}),
		gemini.WithSystemInstruction("You are a helpful assistant. That helps the user translate things."),
		gemini.WithHttpLogging(),
	)
	suite.Require().NoError(err)

	userMsg := llms.NewTextMessage(llms.RoleUser, "My computer said `boop boop boop` when I turned it on. What does that mean?")

	resp, err := client.GenerateStream(ctx, []llms.Message{userMsg}, func(response *llms.Response, err error) bool {
		fmt.Printf("Stream response: %+v\n", response)
		return true
	})
	suite.Require().NoError(err)
	suite.Require().NotEmpty(resp.Message.Parts)

	fmt.Printf("Final response: %+v\n", resp)

	// hasTextPart := false
	hasToolPart := false

	for _, part := range resp.Message.Parts {
		switch p := part.(type) {
		// case llms.TextPart:
		// 	hasTextPart = true
		case llms.ToolCallPart:
			hasToolPart = true

			var params testutil.BoopToolParams
			err := json.Unmarshal(p.Input, &params)
			suite.NoError(err)
			suite.Contains(params.BoopString, "boop boop boop")
		}
	}

	// suite.True(hasTextPart, "response should have a text part")
	suite.True(hasToolPart, "response should have a tool call part")
}

func TestGeminiTestSuite(t *testing.T) {
	suite.Run(t, new(GeminiTestSuite))
}
