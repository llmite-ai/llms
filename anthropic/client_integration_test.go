//go:build integration
// +build integration

package anthropic_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/llmite-ai/llms"
	"github.com/llmite-ai/llms/anthropic"
	"github.com/llmite-ai/llms/testutil"
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

	msg := llms.NewTextMessage(llms.RoleUser, "What is the meaning of life?")

	resp, err := client.Generate(ctx, []llms.Message{msg})
	suite.NoError(err)
	suite.NotEmpty(resp.ID)
	suite.Require().NotEmpty(resp.Message.Parts)

	textPart := resp.Message.Parts[0]
	suite.IsType(llms.TextPart{}, textPart)
}

func (suite *AnthropicTestSuite) TestGenerateSystemPrompt() {
	suite.T().Parallel() // Enable parallel execution for this method
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithHttpLogging(),
	)

	systemMsg := llms.NewTextMessage(llms.RoleSystem, "You only response in 'beep'  Example: beep beep beep.")
	userMsg := llms.NewTextMessage(llms.RoleUser, "Hello, how are you?")

	resp, err := client.Generate(ctx, []llms.Message{systemMsg, userMsg})
	suite.Require().NoError(err)
	suite.NotEmpty(resp.ID)
	suite.Require().NotEmpty(resp.Message.Parts)

	textPart := resp.Message.Parts[0]
	suite.Require().IsType(llms.TextPart{}, textPart)

	text, ok := textPart.(llms.TextPart)
	suite.True(ok)
	suite.Contains(text.Text, "beep")
}

func (suite *AnthropicTestSuite) TestGenerateWithToolCalls() {
	suite.T().Parallel() // Enable parallel execution for this method
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithTools([]llms.Tool{
			testutil.NewBoopTool(),
		}),
		anthropic.WithHttpLogging(),
	)

	systemMsg := llms.NewTextMessage(llms.RoleSystem, "You are a helpful assistant. That helps the user translate things.")
	userMsg := llms.NewTextMessage(llms.RoleUser, "My computer said `boop boop boop` when I turned it on. What does that mean?")

	resp, err := client.Generate(ctx, []llms.Message{systemMsg, userMsg})
	suite.Require().NoError(err)
	suite.NotEmpty(resp.ID)
	suite.Require().NotEmpty(resp.Message.Parts)

	fmt.Printf("Response: %+v\n", resp)

	hasTextPart := false
	hasToolPart := false

	for _, part := range resp.Message.Parts {
		switch p := part.(type) {
		case llms.TextPart:
			hasTextPart = true
		case llms.ToolCallPart:
			hasToolPart = true

			var params testutil.BoopToolParams
			err := json.Unmarshal(p.Input, &params)
			suite.NoError(err)
		}
	}

	suite.True(hasTextPart, "response should have a text part")
	suite.True(hasToolPart, "response should have a tool call part")
}

func (suite *AnthropicTestSuite) TestGenerateStreamBasic() {
	suite.T().Parallel()
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithHttpLogging(),
	)

	msg := llms.NewTextMessage(llms.RoleUser, "Tell me a short story about a robot in exactly 3 sentences.")

	streamCallCount := 0
	var finalResponse *llms.Response

	resp, err := client.GenerateStream(ctx, []llms.Message{msg}, func(response *llms.Response, streamErr error) bool {
		suite.NoError(streamErr, "stream function should not receive errors")
		if response != nil {
			streamCallCount++
			finalResponse = response
			suite.NotEmpty(response.ID)
			suite.Equal(anthropic.ProviderAnthropic, response.Provider)
		}
		return true // Continue streaming
	})

	suite.NoError(err)
	suite.Greater(streamCallCount, 0, "stream function should be called at least once")
	suite.NotNil(resp)
	suite.NotEmpty(resp.ID)
	suite.NotEmpty(resp.Message.Parts)
	suite.Equal(finalResponse.ID, resp.ID, "final response should match last streamed response")

	textPart := resp.Message.Parts[0]
	suite.IsType(llms.TextPart{}, textPart)
}

func (suite *AnthropicTestSuite) TestGenerateStreamSystemPrompt() {
	suite.T().Parallel()
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithHttpLogging(),
	)

	systemMsg := llms.NewTextMessage(llms.RoleSystem, "You only respond with numbers. Count from 1 to 5.")
	userMsg := llms.NewTextMessage(llms.RoleUser, "Start counting")

	streamCallCount := 0
	receivedTexts := make([]string, 0)

	resp, err := client.GenerateStream(ctx, []llms.Message{systemMsg, userMsg}, func(response *llms.Response, streamErr error) bool {
		suite.NoError(streamErr)
		if response != nil && len(response.Message.Parts) > 0 {
			streamCallCount++
			if textPart, ok := response.Message.Parts[0].(llms.TextPart); ok {
				receivedTexts = append(receivedTexts, textPart.Text)
			}
		}
		return true
	})

	suite.NoError(err)
	suite.Greater(streamCallCount, 0)
	suite.NotNil(resp)
	suite.NotEmpty(resp.Message.Parts)

	textPart, ok := resp.Message.Parts[0].(llms.TextPart)
	suite.True(ok)
	
	// Check that the final response contains numbers as expected from system prompt
	suite.Regexp(`[0-9]`, textPart.Text, "response should contain numbers due to system prompt")
}

func (suite *AnthropicTestSuite) TestGenerateStreamWithToolCalls() {
	suite.T().Parallel()
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithTools([]llms.Tool{
			testutil.NewBoopTool(),
		}),
		anthropic.WithHttpLogging(),
	)

	systemMsg := llms.NewTextMessage(llms.RoleSystem, "You are a helpful assistant that translates boops.")
	userMsg := llms.NewTextMessage(llms.RoleUser, "My computer said `boop boop boop`. Please translate this.")

	streamCallCount := 0
	hasSeenToolCall := false

	resp, err := client.GenerateStream(ctx, []llms.Message{systemMsg, userMsg}, func(response *llms.Response, streamErr error) bool {
		suite.NoError(streamErr)
		if response != nil {
			streamCallCount++
			for _, part := range response.Message.Parts {
				if _, ok := part.(llms.ToolCallPart); ok {
					hasSeenToolCall = true
				}
			}
		}
		return true
	})

	suite.NoError(err)
	suite.Greater(streamCallCount, 0)
	suite.NotNil(resp)
	suite.NotEmpty(resp.Message.Parts)

	hasTextPart := false
	hasToolPart := false

	for _, part := range resp.Message.Parts {
		switch p := part.(type) {
		case llms.TextPart:
			hasTextPart = true
		case llms.ToolCallPart:
			hasToolPart = true
			var params testutil.BoopToolParams
			err := json.Unmarshal(p.Input, &params)
			suite.NoError(err)
		}
	}

	suite.True(hasTextPart, "response should have a text part")
	suite.True(hasToolPart, "response should have a tool call part")
	suite.True(hasSeenToolCall, "should have seen tool call during streaming")
}

func (suite *AnthropicTestSuite) TestGenerateStreamEarlyTermination() {
	suite.T().Parallel()
	ctx := context.Background()
	client := anthropic.New(
		anthropic.WithHttpLogging(),
	)

	msg := llms.NewTextMessage(llms.RoleUser, "Write a long story about a journey.")

	streamCallCount := 0
	maxCalls := 3

	resp, err := client.GenerateStream(ctx, []llms.Message{msg}, func(response *llms.Response, streamErr error) bool {
		suite.NoError(streamErr)
		if response != nil {
			streamCallCount++
			if streamCallCount >= maxCalls {
				return false // Stop streaming early
			}
		}
		return true
	})

	suite.NoError(err)
	suite.Equal(maxCalls, streamCallCount, "stream should have been terminated early")
	suite.NotNil(resp)
	suite.NotEmpty(resp.Message.Parts)
}

func (suite *AnthropicTestSuite) TestGenerateStreamErrorHandling() {
	suite.T().Parallel()
	ctx := context.Background()
	
	// Create client with invalid model to potentially trigger errors
	client := anthropic.New(
		anthropic.WithModel("invalid-model-name"),
		anthropic.WithHttpLogging(),
	)

	msg := llms.NewTextMessage(llms.RoleUser, "Hello")

	streamCallCount := 0
	receivedError := false

	resp, err := client.GenerateStream(ctx, []llms.Message{msg}, func(response *llms.Response, streamErr error) bool {
		streamCallCount++
		if streamErr != nil {
			receivedError = true
		}
		return true
	})

	// Expect an error due to invalid model
	suite.Error(err)
	
	// Stream function may or may not be called depending on when the error occurs
	if streamCallCount > 0 {
		suite.True(receivedError, "should have received error in stream function if it was called")
	}
	
	// Response might be nil due to error
	if resp != nil {
		suite.Empty(resp.Message.Parts)
	}
}

func TestAnthropicTestSuite(t *testing.T) {
	suite.Run(t, new(AnthropicTestSuite))
}
