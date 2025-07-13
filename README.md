# llmite

**⚠️ This project is currently in beta. APIs may change.**

llmite is a Go library that provides a unified interface for working with Large Language Models (LLMs) from different providers. It offers a consistent API that allows you to switch between providers like Anthropic Claude and Google Gemini without changing your application code.

## Features

- **Unified Interface**: Single API for multiple LLM providers
- **Provider Support**: Anthropic Claude and Google Gemini
- **Tool Calling**: Built-in support for function calling across providers
- **Streaming**: Real-time response streaming (provider-dependent)
- **HTTP Logging**: Comprehensive request/response logging for debugging
- **Type Safety**: Strongly typed Go interfaces and message structures

## Installation

```bash
go get github.com/jpoz/llmite
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

    "github.com/jpoz/llmite"
    "github.com/jpoz/llmite/anthropic"
)

func main() {
    // Create Anthropic client
    client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
    
    // Create messages
    messages := []llmite.Message{
        {
            Role: llmite.RoleUser,
            Parts: []llmite.Part{
                {Type: llmite.PartTypeText, Text: "What is the capital of France?"},
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

### Basic Usage with Google Gemini

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/jpoz/llmite"
    "github.com/jpoz/llmite/gemini"
)

func main() {
    // Create Gemini client
    client, err := gemini.NewClient(context.Background(), os.Getenv("GEMINI_API_KEY"))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    // Create messages
    messages := []llmite.Message{
        {
            Role: llmite.RoleUser,
            Parts: []llmite.Part{
                {Type: llmite.PartTypeText, Text: "Explain quantum computing in simple terms"},
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

### Streaming Responses

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/jpoz/llmite"
    "github.com/jpoz/llmite/gemini"
)

func main() {
    client, err := gemini.NewClient(context.Background(), os.Getenv("GEMINI_API_KEY"))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()
    
    messages := []llmite.Message{
        {
            Role: llmite.RoleUser,
            Parts: []llmite.Part{
                {Type: llmite.PartTypeText, Text: "Write a short story about a robot"},
            },
        },
    }
    
    // Stream response
    _, err = client.GenerateStream(context.Background(), messages, func(part llmite.Part) error {
        if part.Type == llmite.PartTypeText {
            fmt.Print(part.Text)
        }
        return nil
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

    "github.com/jpoz/llmite"
    "github.com/jpoz/llmite/anthropic"
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
    tools := []llmite.Tool{WeatherTool{}}
    
    messages := []llmite.Message{
        {
            Role: llmite.RoleUser,
            Parts: []llmite.Part{
                {Type: llmite.PartTypeText, Text: "What's the weather like in San Francisco?"},
            },
        },
    }
    
    response, err := client.GenerateWithTools(context.Background(), messages, tools)
    if err != nil {
        log.Fatal(err)
    }
    
    // Handle tool calls
    for _, part := range response.Message.Parts {
        if part.Type == llmite.PartTypeToolCall {
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

    "github.com/jpoz/llmite"
    "github.com/jpoz/llmite/anthropic"
)

func main() {
    // Enable HTTP logging
    llmite.EnableHTTPLogging(llmite.HTTPLoggingConfig{
        LogHeaders:     true,
        LogRequestBody: true,
        LogResponseBody: true,
        MaxBodySize:    1024 * 10, // 10KB
    })
    
    client := anthropic.NewClient(os.Getenv("ANTHROPIC_API_KEY"))
    
    // Your requests will now be logged
    messages := []llmite.Message{
        {
            Role: llmite.RoleUser,
            Parts: []llmite.Part{
                {Type: llmite.PartTypeText, Text: "Hello!"},
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

**Google Gemini:**
- `GEMINI_API_KEY` or `GOOGLE_API_KEY` - Your Google API key
- `GOOGLE_GENAI_USE_VERTEXAI` - Set to "true" to use Vertex AI
- `GOOGLE_CLOUD_PROJECT` - GCP project ID (for Vertex AI)
- `GOOGLE_CLOUD_LOCATION` - GCP location (for Vertex AI)

## Provider Capabilities

| Feature | Anthropic Claude | Google Gemini |
|---------|------------------|---------------|
| Text Generation | ✅ | ✅ |
| Streaming | ❌ | ✅ |
| Tool Calling | ✅ | ✅ |
| System Messages | ✅ | ✅ |
| HTTP Logging | ✅ | ✅ |

## Contributing

This project is in beta. Please report issues and feature requests through GitHub issues.

## License

[Include your license information here]