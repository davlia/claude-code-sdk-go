# Claude Code SDK Go - Transport Implementation

This package provides the transport layer for communicating with the Claude Code CLI.

## Overview

The `SubprocessCLITransport` implements the `Transport` interface to provide communication with Claude Code via subprocess. It supports both string-based prompts and streaming message interactions.

## Features

- **String and Streaming Modes**: Supports both simple string prompts and streaming message interactions
- **Control Requests**: Send interrupt signals and other control commands
- **Session Management**: Maintain conversation context with session IDs
- **Error Handling**: Comprehensive error handling with stderr capture and process management
- **Buffer Management**: JSON accumulation with size limits to handle partial messages
- **MCP Server Support**: Full support for MCP (Model Context Protocol) server configuration

## Usage

### Basic String Prompt

```go
import (
    "github.com/davlia/claude-code-sdk-go"
    "github.com/davlia/claude-code-sdk-go/internal/transport"
)

// Create a string prompt
prompt := claude.NewStringPromptStream("What is 2+2?")
options := claude.NewOptions()

// Create transport
trans := transport.NewSubprocessCLITransport(prompt, options)

// Connect
ctx := context.Background()
if err := trans.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer trans.Disconnect()

// Receive messages
messages := trans.ReceiveMessages(ctx)
for msg := range messages {
    if msg.Err != nil {
        log.Printf("Error: %v", msg.Err)
        continue
    }
    // Process message
    fmt.Printf("Message: %+v\n", msg.Data)
}
```

### Streaming Mode

```go
// Create empty stream for interactive use
prompt := claude.NewEmptyStream()
options := claude.NewOptions()

// Create transport with streaming
trans := transport.NewSubprocessCLITransport(prompt, options).
    WithStreaming(true).
    WithSessionID("my-session")

// Connect
if err := trans.Connect(ctx); err != nil {
    log.Fatal(err)
}
defer trans.Disconnect()

// Send messages
messages := []map[string]any{
    {
        "type": "user",
        "message": map[string]any{
            "role":    "user",
            "content": "Hello!",
        },
    },
}

if err := trans.SendRequest(ctx, messages, nil); err != nil {
    log.Fatal(err)
}

// Receive responses
for msg := range trans.ReceiveMessages(ctx) {
    // Process messages
}
```

### Configuration Options

The transport supports all Claude Code CLI options:

```go
options := claude.NewOptions()
options.SystemPrompt = "You are a helpful assistant"
options.Model = "claude-3-sonnet"
options.AllowedTools = []string{"read", "write", "execute"}
options.DisallowedTools = []string{"dangerous_tool"}
options.PermissionMode = claude.PermissionModeAcceptEdits
options.Cwd = "/path/to/working/directory"

// MCP Server configuration
options.MCPServers = map[string]claude.MCPServerConfig{
    "myserver": claude.MCPStdioServerConfig{
        Command: "mcp-server",
        Args:    []string{"--port", "8080"},
        Env:     map[string]string{"API_KEY": "secret"},
    },
}
```

### Builder Methods

The transport provides several builder methods for configuration:

```go
trans := transport.NewSubprocessCLITransport(prompt, options).
    WithCLIPath("/custom/path/to/claude").        // Custom CLI path
    WithStreaming(true).                          // Enable streaming mode
    WithSessionID("my-session").                  // Set session ID
    WithCloseStdinAfterPrompt(true)             // Close stdin after sending prompt
```

## CLI Discovery

The transport automatically discovers the Claude Code CLI in the following order:

1. `CLAUDE_CODE_CLI_PATH` environment variable
2. System PATH (`claude` command)
3. Common installation locations:
   - `~/.npm-global/bin/claude`
   - `/usr/local/bin/claude`
   - `~/.local/bin/claude`
   - `~/node_modules/.bin/claude`
   - `~/.yarn/bin/claude`
   - Windows: `%APPDATA%\npm\claude`

## Error Handling

The transport provides detailed error types:

- `CLINotFoundError`: Claude Code CLI not found
- `CLIConnectionError`: Connection issues
- `ProcessError`: CLI process failures with exit codes
- `CLIJSONDecodeError`: JSON parsing errors

## Thread Safety

The transport is thread-safe and can handle concurrent operations:
- Multiple goroutines can read from `ReceiveMessages()` channel
- `SendRequest()` is protected by internal synchronization
- Connection state is managed with proper locking

## Testing

Run integration tests with:

```bash
CLAUDE_INTEGRATION_TEST=1 go test ./internal/transport/...
```

## Implementation Details

### JSON Buffer Management

The transport accumulates partial JSON messages up to 1MB to handle streaming responses that may be split across multiple lines.

### Process Lifecycle

1. **Connect**: Starts the CLI subprocess with proper pipes
2. **Communication**: Handles stdin/stdout/stderr streams
3. **Disconnect**: Gracefully shuts down with timeout and force-kill fallback

### Control Requests

In streaming mode, the transport supports control requests like interrupts:

```go
// Send interrupt signal
if err := trans.Interrupt(ctx); err != nil {
    log.Printf("Failed to interrupt: %v", err)
}
```

## Differences from Python SDK

This Go implementation provides equivalent functionality to the Python SDK's `SubprocessCLITransport` with Go-idiomatic patterns:

- Uses channels instead of async iterators
- Goroutines for concurrent stream handling
- Context-based cancellation
- Builder pattern for configuration
- Strong typing with proper interfaces