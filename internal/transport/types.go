package transport

import "context"

// MessageStream represents a stream of messages
type MessageStream interface {
	// Next returns the next message in the stream
	Next(ctx context.Context) (map[string]any, error)
}

// Options represents configuration options for the transport
type Options struct {
	// Model to use (e.g., "claude-3-opus-20240229")
	Model string
	
	// SystemPrompt is the initial system message
	SystemPrompt string
	
	// AppendSystemPrompt appends additional text to the system prompt
	AppendSystemPrompt string
	
	// Working directory for the subprocess
	Cwd string
	
	// Allowed tools
	AllowedTools []string
	
	// Disallowed tools
	DisallowedTools []string
	
	// Max conversation turns
	MaxTurns *int
	
	// Permission prompt tool name
	PermissionPromptToolName string
	
	// Permission mode
	PermissionMode string
	
	// Continue conversation from previous session
	ContinueConversation bool
	
	// Resume conversation ID
	Resume string
	
	// MCP server configurations
	MCPServers map[string]any
}

// NewOptions creates a new Options with defaults
func NewOptions() *Options {
	return &Options{}
}