# Architecture Documentation

This directory contains architectural documentation for the Claude Code SDK for Go.

## Overview

The SDK is structured following clean architecture principles with clear separation between public API and internal implementation details.

## Package Structure

```
github.com/davlia/claude-code-sdk-go/
├── Public API (exported)
│   ├── client.go      - Bidirectional client
│   ├── query.go       - Stateless query function
│   ├── types.go       - Public types
│   ├── options.go     - Configuration options
│   ├── errors.go      - Error types
│   └── version.go     - Version information
│
└── internal/          - Implementation details (not exported)
    ├── transport/     - CLI communication layer
    └── parser/        - Message parsing logic
```

## Design Principles

1. **Minimal Public API**: Only expose what users need
2. **Clean Separation**: Internal packages hide implementation details
3. **Testability**: All components are easily testable
4. **Extensibility**: Easy to add new features without breaking changes
5. **Error Handling**: Comprehensive error types for all failure modes

## Key Components

### Transport Layer
The transport layer handles communication with the Claude Code CLI through subprocess management.

### Parser
The parser converts raw JSON messages into strongly-typed Go structures.

### Client
The client provides a stateful, bidirectional interface for interactive conversations.

### Query Function
A simple, stateless function for one-shot queries without connection management.