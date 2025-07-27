package claude

import (
	"context"
)

// MessageStream represents a stream of messages
type MessageStream interface {
	// Next returns the next message in the stream
	Next(ctx context.Context) (map[string]any, error)
}

// StringPrompt wraps a string prompt as a MessageStream
type stringPrompt struct {
	prompt string
	sent   bool
}

func (s *stringPrompt) Next(ctx context.Context) (map[string]any, error) {
	if s.sent {
		return nil, nil // EOF
	}
	s.sent = true
	return map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": s.prompt,
		},
		"parent_tool_use_id": nil,
		"session_id":         "default",
	}, nil
}

// Query sends a query to Claude Code and returns a channel of messages.
//
// This function is ideal for simple, stateless queries where you don't need
// bidirectional communication or conversation management. For interactive,
// stateful conversations, use Client instead.
//
// Key differences from Client:
// - Unidirectional: Send all messages upfront, receive all responses
// - Stateless: Each query is independent, no conversation state
// - Simple: Fire-and-forget style, no connection management
// - No interrupts: Cannot interrupt or send follow-up messages
//
// When to use Query:
// - Simple one-off questions ("What is 2+2?")
// - Batch processing of independent prompts
// - Code generation or analysis tasks
// - Automated scripts and CI/CD pipelines
// - When you know all inputs upfront
//
// When to use Client:
// - Interactive conversations with follow-ups
// - Chat applications or REPL-like interfaces
// - When you need to send messages based on responses
// - When you need interrupt capabilities
// - Long-running sessions with state
//
// Parameters:
//   - ctx: Context for cancellation
//   - prompt: The prompt to send to Claude. Can be a string for single-shot queries
//     or a MessageStream for streaming mode with continuous interaction.
//   - options: Optional configuration (defaults to NewOptions() if nil).
//     Set options.PermissionMode to control tool execution:
//   - PermissionModeDefault: CLI prompts for dangerous tools
//   - PermissionModeAcceptEdits: Auto-accept file edits
//   - PermissionModeBypassPermissions: Allow all tools (use with caution)
//     Set options.Cwd for working directory.
//
// Returns:
//   - A channel that yields messages from the conversation
//   - An error if the query cannot be initiated
//
// Example - Simple query:
//
//	messages, err := Query(ctx, "What is the capital of France?", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for msg := range messages {
//	    if msg.Error != nil {
//	        log.Fatal(msg.Error)
//	    }
//	    fmt.Println(msg.Message)
//	}
//
// Example - With options:
//
//	options := &Options{
//	    SystemPrompt: "You are an expert Python developer",
//	    Cwd: "/home/user/project",
//	}
//	messages, err := Query(ctx, "Create a Python web server", options)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for msg := range messages {
//	    // Process messages...
//	}
func Query(ctx context.Context, prompt any, options *Options) (<-chan MessageResult, error) {
	// Create a client with the given options
	client := NewClient(options)
	
	// Connect with the prompt
	if err := client.Connect(ctx, prompt); err != nil {
		return nil, err
	}
	
	// Create output channel
	out := make(chan MessageResult)
	
	// Start goroutine to receive messages and auto-disconnect when done
	go func() {
		defer close(out)
		defer client.Disconnect()
		
		// Receive all messages until completion
		for msg := range client.ReceiveMessages(ctx) {
			out <- msg
			
			// If there's an error or we received a result message, we're done
			if msg.Error != nil {
				return
			}
			if _, isResult := msg.Message.(*ResultMessage); isResult {
				return
			}
		}
	}()
	
	return out, nil
}
