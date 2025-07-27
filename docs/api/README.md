# API Documentation

This directory contains detailed API documentation for the Claude Code SDK for Go.

## Contents

- [Client API](client.md) - Bidirectional client for interactive conversations
- [Query API](query.md) - Simple stateless query function
- [Types](types.md) - Data types and structures
- [Options](options.md) - Configuration options
- [Errors](errors.md) - Error types and handling

## Quick Start

For simple one-shot queries:

```go
messages, err := claude.Query(ctx, "What is 2+2?", nil)
```

For interactive conversations:

```go
client := claude.NewClient(nil)
err := client.Connect(ctx, nil)
// ... send and receive messages ...
client.Disconnect()
```

See the [examples](../examples/) directory for more detailed usage.