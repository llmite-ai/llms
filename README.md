# llms

**⚠️ This project is currently in beta. APIs may change.**

llms is a Go library that provides a unified interface for working with Large Language Models (LLMs) from different providers. It offers a consistent API that allows you to switch between providers like Anthropic Claude and Google Gemini without changing your application code.

## Features

- **Unified Interface**: Single API for multiple LLM providers
- **Provider Support**: Anthropic Claude, Google Gemini, and OpenAI
- **Tool Calling**: Built-in support for function calling across providers
- **Streaming**: Real-time response streaming (provider-dependent)
- **HTTP Logging**: Comprehensive request/response logging for debugging
- **Type Safety**: Strongly typed Go interfaces and message structures

## Installation

```bash
go get github.com/llmite-ai/llms
```

## Quick Start

### Basic Usage with Anthropic Claude

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/llmite-ai/llms"
    "github.com/llmite-ai/llms/anthropic"
)

func main() {
    // Create Anthropic client
    client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
    
    // Create messages
    messages := []llms.Message{
        {
            Role: llms.RoleUser,
            Parts: []llms.Part{
                {Type: llms.PartTypeText, Text: "What is the capital of France?"},
            },
        },
    }
    
    // Generate response
    response, err := client.Generate(context.Background(), messages)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println(response.Message.Parts[0].Text)
}
```

### Basic Usage with OpenAI

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/llmite-ai/llms"
    "github.com/llmite-ai/llms/openai"
)

func main() {
    // Create OpenAI client
    client := openai.New(
        openai.WithModel("gpt-4o-mini"),
        openai.WithMaxTokens(100),
    )
    
    // Create messages
    messages := []llms.Message{
        {
            Role: llms.RoleUser,
            Parts: []llms.Part{
                llms.TextPart{Text: "What is the capital of France?"},
            },
        },
    }
    
    // Generate response
    response, err := client.Generate(context.Background(), messages)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, part := range response.Message.Parts {
        if textPart, ok := part.(llms.TextPart); ok {
            fmt.Println(textPart.Text)
        }
    }
}
```

### Basic Usage with Google Gemini

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/llmite-ai/llms"
    "github.com/llmite-ai/llms/gemini"
)

func main() {
    // Create Gemini client
    client, err := gemini.New(
        gemini.WithApiKey(os.Getenv("GEMINI_API_KEY")),
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Create messages
    messages := []llms.Message{
        {
            Role: llms.RoleUser,
            Parts: []llms.Part{
                llms.TextPart{Text: "Explain quantum computing in simple terms"},
            },
        },
    }
    
    // Generate response
    response, err := client.Generate(context.Background(), messages)
    if err != nil {
        log.Fatal(err)
    }
    
    for _, part := range response.Message.Parts {
        if textPart, ok := part.(llms.TextPart); ok {
            fmt.Println(textPart.Text)
        }
    }
}
```

### Streaming Responses

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/llmite-ai/llms"
    "github.com/llmite-ai/llms/openai"
)

func main() {
    client := openai.New(
        openai.WithModel("gpt-4o-mini"),
    )
    
    messages := []llms.Message{
        {
            Role: llms.RoleUser,
            Parts: []llms.Part{
                llms.TextPart{Text: "Write a short story about a robot"},
            },
        },
    }
    
    // Stream response
    _, err := client.GenerateStream(context.Background(), messages, func(response *llms.Response, err error) bool {
        if err != nil {
            log.Printf("Streaming error: %v", err)
            return false
        }
        
        // Process each part in the response
        for _, part := range response.Message.Parts {
            if textPart, ok := part.(llms.TextPart); ok {
                fmt.Print(textPart.Text)
            }
        }
        
        return true // Continue streaming
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Using Tools/Function Calling

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/llmite-ai/llms"
    "github.com/llmite-ai/llms/anthropic"
)

// Define a simple tool
type WeatherTool struct{}

func (w WeatherTool) Name() string {
    return "get_weather"
}

func (w WeatherTool) Description() string {
    return "Get current weather for a location"
}

type WeatherParams struct {
    Location string `json:"location" jsonschema:"description=The city or location to get weather for"`
}

func (w WeatherTool) Parameters() interface{} {
    return WeatherParams{}
}

func (w WeatherTool) Execute(input string) (string, error) {
    // In a real implementation, you'd call a weather API
    return fmt.Sprintf("The weather in %s is sunny, 72°F", input), nil
}

func main() {
    client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
    
    // Register tools
    tools := []llms.Tool{WeatherTool{}}
    
    messages := []llms.Message{
        {
            Role: llms.RoleUser,
            Parts: []llms.Part{
                {Type: llms.PartTypeText, Text: "What's the weather like in San Francisco?"},
            },
        },
    }
    
    response, err := client.GenerateWithTools(context.Background(), messages, tools)
    if err != nil {
        log.Fatal(err)
    }
    
    // Handle tool calls
    for _, part := range response.Message.Parts {
        if part.Type == llms.PartTypeToolCall {
            fmt.Printf("Tool called: %s\n", part.ToolCall.Name)
            // Execute tool and add result to conversation...
        }
    }
}
```

### HTTP Logging for Debugging

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/llmite-ai/llms"
    "github.com/llmite-ai/llms/anthropic"
)

func main() {
    // Enable HTTP logging
    llms.EnableHTTPLogging(llms.HTTPLoggingConfig{
        LogHeaders:     true,
        LogRequestBody: true,
        LogResponseBody: true,
        MaxBodySize:    1024 * 10, // 10KB
    })
    
    client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
    
    // Your requests will now be logged
    messages := []llms.Message{
        {
            Role: llms.RoleUser,
            Parts: []llms.Part{
                {Type: llms.PartTypeText, Text: "Hello!"},
            },
        },
    }
    
    response, err := client.Generate(context.Background(), messages)
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println(response.Message.Parts[0].Text)
}
```

## Configuration

### Environment Variables

**Anthropic:**
- `ANTHROPIC_API_KEY` - Your Anthropic API key
- `ANTHROPIC_AUTH_TOKEN` - Alternative auth token
- `ANTHROPIC_BASE_URL` - Custom base URL (optional)

**OpenAI:**
- `OPENAI_API_KEY` - Your OpenAI API key

**Google Gemini:**
- `GEMINI_API_KEY` or `GOOGLE_API_KEY` - Your Google API key
- `GOOGLE_GENAI_USE_VERTEXAI` - Set to "true" to use Vertex AI
- `GOOGLE_CLOUD_PROJECT` - GCP project ID (for Vertex AI)
- `GOOGLE_CLOUD_LOCATION` - GCP location (for Vertex AI)

## Provider Capabilities

| Feature | Anthropic Claude | Google Gemini | OpenAI |
|---------|------------------|---------------|--------|
| Text Generation | ✅ | ✅ | ✅ |
| Streaming | ❌ | ✅ | ✅ |
| Tool Calling | ✅ | ✅ | 🚧* |
| System Messages | ✅ | ✅ | ✅ |
| HTTP Logging | ✅ | ✅ | ✅ |

*🚧 = Partially implemented or in progress

## Contributing

This project is in beta. Please report issues and feature requests through GitHub issues.

## License

[Include your license information here]
