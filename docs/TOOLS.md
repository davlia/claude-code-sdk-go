# Claude Code SDK for Go - Tools Guide

## Overview

Claude can use various tools to perform actions like reading files, searching code, running commands, and more. This guide explains how to work with tools in the Claude Code SDK for Go.

## Available Tools

Claude has access to the following tools by default:

- **File Operations**: Read, write, and edit files
- **Search Tools**: Search for patterns in code
- **Command Execution**: Run shell commands
- **Web Tools**: Fetch web content and search
- **Code Analysis**: Analyze and understand codebases

## Tool Usage in Responses

When Claude uses tools, the response will contain tool use blocks:

```go
result, err := claude.Query(ctx, "Read the README.md file")
if err != nil {
    log.Fatal(err)
}

// The result will contain tool use information
// Claude will automatically use the Read tool to read the file
fmt.Println(result.Content)
```

## MCP (Model Context Protocol) Servers

MCP servers extend Claude's capabilities with additional tools. You can configure MCP servers when creating a client:

### Filesystem Server

Provides file system access tools:

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

client, err := claude.NewClient(ctx, "Help me work with files", options)
```

### GitHub Server

Provides GitHub integration:

```go
options := &claude.Options{
    MCPServers: []claude.MCPServerConfig{
        {
            Name: "github",
            Type: claude.MCPServerTypeStdio,
            Config: map[string]interface{}{
                "command": "npx",
                "args":    []string{"-y", "@modelcontextprotocol/server-github"},
                "env": map[string]string{
                    "GITHUB_TOKEN": os.Getenv("GITHUB_TOKEN"),
                },
            },
        },
    },
}
```

### Postgres Server

Provides database access:

```go
options := &claude.Options{
    MCPServers: []claude.MCPServerConfig{
        {
            Name: "postgres",
            Type: claude.MCPServerTypeStdio,
            Config: map[string]interface{}{
                "command": "npx",
                "args":    []string{"-y", "@modelcontextprotocol/server-postgres"},
                "env": map[string]string{
                    "DATABASE_URL": os.Getenv("DATABASE_URL"),
                },
            },
        },
    },
}
```

## Working with Tool Results

When Claude uses tools, you can see the tool usage in the response:

```go
// Interactive example showing tool usage
client, err := claude.NewClient(ctx, nil)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Ask Claude to use tools
result, err := client.SendMessage(ctx, claude.UserMessage(
    "Create a new file called hello.txt with 'Hello, World!' content",
))
if err != nil {
    log.Fatal(err)
}

// Claude will use the Write tool automatically
fmt.Println("Claude:", result.Content)
```

## Best Practices

### 1. Tool Selection
Claude automatically selects the appropriate tools based on the task. You don't need to specify which tools to use.

### 2. Error Handling
Always handle errors when Claude uses tools:

```go
result, err := claude.Query(ctx, "Run a system command")
if err != nil {
    // Handle tool execution errors
    log.Printf("Tool error: %v", err)
}
```

### 3. Resource Management
When using MCP servers, ensure proper cleanup:

```go
client, err := claude.NewClient(ctx, nil, options)
if err != nil {
    log.Fatal(err)
}
defer client.Close() // Ensures MCP servers are properly shut down
```

### 4. Security Considerations

- Be cautious when allowing file system or command execution access
- Use appropriate permissions and sandboxing
- Validate tool inputs and outputs
- Consider using read-only MCP servers when write access isn't needed

## Custom MCP Servers

You can create custom MCP servers to extend Claude's capabilities:

```go
options := &claude.Options{
    MCPServers: []claude.MCPServerConfig{
        {
            Name: "custom-tools",
            Type: claude.MCPServerTypeStdio,
            Config: map[string]interface{}{
                "command": "./my-custom-mcp-server",
                "args":    []string{"--mode", "production"},
            },
        },
    },
}
```

## Examples

### File Analysis
```go
result, err := claude.Query(ctx, 
    "Analyze all Python files in the src directory and summarize their purpose",
)
```

### Code Generation
```go
result, err := claude.Query(ctx,
    "Create a REST API server in Go with endpoints for user management",
)
```

### System Information
```go
result, err := claude.Query(ctx,
    "Check the system's CPU and memory usage",
)
```

Each of these requests will cause Claude to use appropriate tools automatically to complete the task.