package transport

import (
	"context"
)

// Transport defines the interface for communication with Claude Code CLI.
type Transport interface {
	// Connect establishes a connection to the Claude Code CLI.
	Connect(ctx context.Context) error
	
	// Disconnect terminates the connection.
	Disconnect() error
	
	// ReceiveMessages returns a channel that yields messages from Claude.
	ReceiveMessages(ctx context.Context) <-chan MessageData
	
	// SendRequest sends additional messages (only works in streaming mode).
	SendRequest(ctx context.Context, messages []map[string]any, metadata map[string]any) error
	
	// Interrupt sends an interrupt signal.
	Interrupt(ctx context.Context) error
	
	// IsConnected checks if the transport is connected.
	IsConnected() bool
}