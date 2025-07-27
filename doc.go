// Package claude provides a Go SDK for interacting with Claude through the Claude Code CLI.
//
// The SDK offers two main ways to interact with Claude:
//
// 1. Query function for simple, stateless queries
// 2. Client type for interactive, stateful conversations
//
// Basic usage:
//
//	import "github.com/davlia/claude-code-sdk-go"
//
//	// Simple query
//	resp, err := claude.Query(ctx, "What is the capital of France?")
//
//	// Interactive client
//	client, err := claude.NewClient()
//	defer client.Close()
//	resp, err := client.SendMessage(ctx, claude.UserMessage("Hello, Claude!"))
package claude
