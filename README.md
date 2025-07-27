# Claude Code SDK for Go

Go SDK for Claude Code. See the [Claude Code SDK documentation](https://docs.anthropic.com/en/docs/claude-code/sdk) for more information.

## Installation

```bash
go get github.com/davlia/claude-code-sdk-go
```

**Prerequisites:**
- Go 1.21+
- Node.js 
- Claude Code: `npm install -g @anthropic-ai/claude-code`

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    claude "github.com/davlia/claude-code-sdk-go"
)

func main() {
    ctx := context.Background()
    
    messages, err := claude.Query(ctx, "What is 2 + 2?", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    for msg := range messages {
        if msg.Error != nil {
            log.Fatal(msg.Error)
        }
        fmt.Println(msg.Message)
    }
}
```

## Usage

### Basic Query

```go
import claude "github.com/davlia/claude-code-sdk-go"

// Simple query
messages, err := claude.Query(ctx, "Hello Claude", nil)
if err != nil {
    log.Fatal(err)
}

for msg := range messages {
    if assistantMsg, ok := msg.Message.(*claude.AssistantMessage); ok {
        for _, block := range assistantMsg.Content {
            if textBlock, ok := block.(*claude.TextBlock); ok {
                fmt.Println(textBlock.Text)
            }
        }
    }
}

// With options
options := &claude.Options{
    SystemPrompt: "You are a helpful assistant",
    MaxTurns:     1,
}

messages, err = claude.Query(ctx, "Tell me a joke", options)
```

### Using Tools

```go
options := &claude.Options{
    AllowedTools:    []string{"Read", "Write", "Bash"},
    PermissionMode: claude.PermissionModeAcceptEdits, // auto-accept file edits
}

messages, err := claude.Query(ctx, "Create a hello.go file", options)
if err != nil {
    log.Fatal(err)
}

// Process tool use and results
for msg := range messages {
    // Handle messages...
}
```

### Working Directory

```go
options := &claude.Options{
    Cwd: "/path/to/project",
}
```

## Interactive Client

For interactive, bidirectional conversations:

```go
client := claude.NewClient(nil)

// Connect (automatically connects with empty stream)
err := client.Connect(ctx, nil)
if err != nil {
    log.Fatal(err)
}
defer client.Disconnect()

// Send a message
err = client.Query(ctx, "Let's solve a math problem step by step", "default")
if err != nil {
    log.Fatal(err)
}

// Receive messages
messages := client.ReceiveMessages(ctx)
for msg := range messages {
    if assistantMsg, ok := msg.Message.(*claude.AssistantMessage); ok {
        // Handle assistant response
    }
}

// Send follow-up
err = client.Query(ctx, "What's 15% of 80?", "default")
```

## API Reference

### `Query(ctx, prompt, options) (<-chan MessageResult, error)`

Main function for querying Claude.

**Parameters:**
- `ctx` (context.Context): Context for cancellation
- `prompt` (string | MessageStream): The prompt to send to Claude
- `options` (*Options): Optional configuration

**Returns:** Channel of MessageResult containing messages or errors

### Types

See the package documentation for complete type definitions:
- `Options` - Configuration options
- `AssistantMessage`, `UserMessage`, `SystemMessage`, `ResultMessage` - Message types
- `TextBlock`, `ToolUseBlock`, `ToolResultBlock` - Content blocks

## Error Handling

```go
messages, err := claude.Query(ctx, "Hello", nil)
if err != nil {
    switch e := err.(type) {
    case *claude.CLINotFoundError:
        log.Fatal("Please install Claude Code")
    case *claude.ProcessError:
        log.Fatalf("Process failed with exit code: %d", e.ExitCode)
    case *claude.CLIJSONDecodeError:
        log.Fatalf("Failed to parse response: %v", e)
    default:
        log.Fatal(err)
    }
}
```

## Available Tools

See the [Claude Code documentation](https://docs.anthropic.com/en/docs/claude-code/settings#tools-available-to-claude) for a complete list of available tools.

## Examples

See [examples/](examples/) for complete working examples.

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT