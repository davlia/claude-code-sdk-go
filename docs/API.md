# Claude Code SDK for Go - API Reference

## Overview

The Claude Code SDK for Go provides a simple interface to interact with Claude through the Claude Code CLI.

## Installation

```bash
go get github.com/davlia/claude-code-sdk-go
```

## Core Functions

### Query

```go
func Query(ctx context.Context, prompt interface{}, options ...*Options) (*Result, error)
```

Performs a one-shot query to Claude. This is the simplest way to interact with Claude.

**Parameters:**
- `ctx`: Context for cancellation and timeout control
- `prompt`: String or MessageStream containing the query
- `options`: Optional configuration settings

**Returns:**
- `*Result`: The response from Claude
- `error`: Any error that occurred

**Example:**
```go
result, err := claude.Query(ctx, "What is the capital of France?")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Content)
```

## Types

### Client

```go
type Client struct {
    // unexported fields
}
```

A client for interactive conversations with Claude.

#### Methods

##### NewClient
```go
func NewClient(ctx context.Context, prompt interface{}, options ...*Options) (*Client, error)
```

Creates a new client for interactive conversations.

##### SendMessage
```go
func (c *Client) SendMessage(ctx context.Context, message interface{}) (*Result, error)
```

Sends a message to Claude and returns the response.

##### Results
```go
func (c *Client) Results() <-chan *Result
```

Returns a channel that receives results from Claude.

##### Close
```go
func (c *Client) Close() error
```

Closes the client connection.

### Options

```go
type Options struct {
    Model               *string
    MaxTokens           *int
    Temperature         *float64
    APIKey              *string
    BaseURL             *string
    BaseDelay           *string
    MaxRetries          *int
    Timeout             *string
    MCPServers          []MCPServerConfig
}
```

Configuration options for Claude interactions.

### Result

```go
type Result struct {
    Content string
    Model   string
    Role    string
    Type    string
    // Additional fields
}
```

Represents a response from Claude.

### Message Types

#### UserMessage
```go
func UserMessage(content string) map[string]interface{}
```

Creates a user message.

#### AssistantMessage
```go
func AssistantMessage(content string) map[string]interface{}
```

Creates an assistant message.

#### SystemMessage
```go
func SystemMessage(content string) map[string]interface{}
```

Creates a system message.

## Error Types

### ClaudeSDKError
Base error type for all SDK errors.

### CLIConnectionError
Returned when unable to connect to Claude Code CLI.

### CLINotFoundError
Returned when Claude Code CLI is not found.

### ProcessError
Returned when the CLI process fails.

### CLIJSONDecodeError
Returned when unable to decode JSON from CLI output.

### MessageParseError
Returned when unable to parse a message from CLI output.

## Advanced Usage

### Custom Message Streams

```go
type customStream struct {
    messages []map[string]interface{}
    index    int
}

func (s *customStream) Next(ctx context.Context) (map[string]interface{}, error) {
    if s.index >= len(s.messages) {
        return nil, nil // End of stream
    }
    msg := s.messages[s.index]
    s.index++
    return msg, nil
}

// Use with Client
stream := &customStream{messages: myMessages}
client, err := claude.NewClient(ctx, stream)
```

### MCP Server Configuration

```go
options := &claude.Options{
    MCPServers: []claude.MCPServerConfig{
        {
            Name: "filesystem",
            Type: claude.MCPServerTypeStdio,
            Config: map[string]interface{}{
                "command": "npx",
                "args":    []string{"-y", "@modelcontextprotocol/server-filesystem"},
            },
        },
    },
}

result, err := claude.Query(ctx, "List files in current directory", options)
```