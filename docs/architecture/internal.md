# Internal Implementation Details

This document describes the internal implementation of the Claude Code SDK for Go.

## Transport Layer

The transport layer (transport.go) handles communication with the Claude Code CLI through subprocess management. This is marked as internal through lowercase naming conventions rather than package separation to avoid import cycles.

### Key Components

- `transport` interface - Defines the contract for CLI communication
- `subprocessCLITransport` - Implementation using subprocess execution
- `messageData` - Internal message wrapper type

All of these are unexported (lowercase) to keep them internal to the package.

## Message Parsing

Message parsing is handled in types.go alongside the type definitions. The parsing functions are unexported:

- `parseMessage` - Main parser entry point
- `parseUserMessage` - Parses user messages
- `parseAssistantMessage` - Parses assistant messages with content blocks
- `parseSystemMessage` - Parses system messages
- `parseResultMessage` - Parses result messages

## Design Decisions

### Why Not Use internal/ Package?

Initially, we attempted to use Go's `internal/` package structure, but this created import cycles because:

1. The transport needs access to types like `Options` and `MessageStream`
2. The parser needs access to all the message types
3. The main package needs the transport and parser

To avoid these cycles while maintaining clear separation, we use naming conventions:
- Exported (uppercase) = Public API
- Unexported (lowercase) = Internal implementation

This is a common pattern in Go standard library packages.