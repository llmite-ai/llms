# LLMite Examples

This directory contains examples showing how to use the LLMite library with different providers.

## Anthropic Examples

### Basic Usage (`basic_anthropic.go`)

Shows how to use the Anthropic client for simple request/response interactions:

```bash
# Set your API key
export ANTHROPIC_API_KEY="your-api-key-here"

# Run the example
go run basic_anthropic.go
```

Features demonstrated:
- Creating an Anthropic client with custom model and max tokens
- Sending system and user messages
- Handling the response and extracting text parts

### Streaming (`streaming_anthropic.go`)

Shows how to use streaming responses with the Anthropic client:

```bash
# Set your API key
export ANTHROPIC_API_KEY="your-api-key-here"

# Run the example
go run streaming_anthropic.go
```

Features demonstrated:
- Creating an Anthropic client for streaming
- Implementing a stream function to handle real-time responses
- Processing streaming chunks and displaying them live
- Handling the final response

## Environment Variables

The examples use the following environment variables:

- `ANTHROPIC_API_KEY` or `ANTHROPIC_AUTH_TOKEN` - Your Anthropic API key
- `ANTHROPIC_BASE_URL` - Optional custom base URL (defaults to https://api.anthropic.com)

## Key Features

- **Model Selection**: Examples use `claude-sonnet-4-20250514` but you can change to any supported model
- **Token Limits**: Configurable max tokens for responses
- **Error Handling**: Proper error handling for API failures
- **Response Types**: Support for text responses and tool calls
- **Streaming**: Real-time response processing with user-defined stream functions

## Running the Examples

Make sure you have Go installed and your `ANTHROPIC_API_KEY` set:

```bash
cd _examples
go mod init examples
go mod tidy
go run basic_anthropic.go
go run streaming_anthropic.go
```

Each example is self-contained and demonstrates different aspects of the LLMite library.