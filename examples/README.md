# Claude Code SDK for Go - Examples

This directory contains example applications demonstrating how to use the Claude Code SDK for Go.

## Examples

### quick_start
A simple example showing basic usage of the SDK with different options.

```bash
cd quick_start
go run main.go
```

### interactive_client
Demonstrates the interactive Client API for bidirectional conversations.

```bash
cd interactive_client
go run main.go
```

### streaming_mode
Shows how to use custom message streams for batch processing.

```bash
cd streaming_mode
go run main.go
```

## Building All Examples

From the project root:

```bash
make examples
```

## Running All Examples

From the project root:

```bash
make run-examples
```

Note: You need to have Claude Code installed (`npm install -g @anthropic-ai/claude-code`) to run these examples.