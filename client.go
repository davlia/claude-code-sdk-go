package claude

import (
	"context"
	"os"
	"sync"
	
	"github.com/davlia/claude-code-sdk-go/internal/transport"
)

// Client provides bidirectional, interactive conversations with Claude Code.
//
// This client provides full control over the conversation flow with support
// for streaming, interrupts, and dynamic message sending. For simple one-shot
// queries, consider using the Query function instead.
//
// Key features:
// - Bidirectional: Send and receive messages at any time
// - Stateful: Maintains conversation context across messages
// - Interactive: Send follow-ups based on responses
// - Control flow: Support for interrupts and session management
//
// When to use Client:
// - Building chat interfaces or conversational UIs
// - Interactive debugging or exploration sessions
// - Multi-turn conversations with context
// - When you need to react to Claude's responses
// - Real-time applications with user input
// - When you need interrupt capabilities
//
// When to use Query instead:
// - Simple one-off questions
// - Batch processing of prompts
// - Fire-and-forget automation scripts
// - When all inputs are known upfront
// - Stateless operations
//
// Example - Interactive conversation:
//
//	client := NewClient(nil)
//	err := client.Connect(ctx, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Disconnect()
//
//	// Send initial message
//	err = client.Query(ctx, "Let's solve a math problem step by step", "default")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Receive and process response
//	messages := client.ReceiveMessages(ctx)
//	for msg := range messages {
//	    // Process messages...
//	}
//
//	// Send follow-up based on response
//	err = client.Query(ctx, "What's 15% of 80?", "default")
type Client struct {
	options   *Options
	transport transport.Transport
	mu        sync.Mutex
}

// NewClient creates a new Claude SDK client
func NewClient(options *Options) *Client {
	if options == nil {
		options = NewOptions()
	}
	os.Setenv("CLAUDE_CODE_ENTRYPOINT", "sdk-go-client")
	return &Client{
		options: options,
	}
}

// Connect establishes a connection to Claude with an optional prompt or message stream.
// If prompt is nil, connects with an empty stream for interactive use.
func (c *Client) Connect(ctx context.Context, prompt any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.transport != nil {
		return NewCLIConnectionError("Already connected")
	}

	// Convert prompt to MessageStream
	var stream MessageStream
	switch p := prompt.(type) {
	case nil:
		// Empty stream for interactive use
		stream = &emptyStream{}
	case string:
		stream = &stringPrompt{prompt: p}
	case MessageStream:
		stream = p
	default:
		return &SDKError{message: "prompt must be nil, a string, or MessageStream"}
	}

	// Convert claude.Options to transport.Options
	transportOptions := &transport.Options{
		Model:                    c.options.Model,
		SystemPrompt:             c.options.SystemPrompt,
		AppendSystemPrompt:       c.options.AppendSystemPrompt,
		Cwd:                      c.options.Cwd,
		AllowedTools:             c.options.AllowedTools,
		DisallowedTools:          c.options.DisallowedTools,
		MaxTurns:                 c.options.MaxTurns,
		PermissionPromptToolName: c.options.PermissionPromptToolName,
		PermissionMode:           string(c.options.PermissionMode),
		ContinueConversation:     c.options.ContinueConversation,
		Resume:                   c.options.Resume,
	}
	
	// Convert MCPServers if present
	if c.options.MCPServers != nil {
		transportOptions.MCPServers = make(map[string]any)
		for k, v := range c.options.MCPServers {
			transportOptions.MCPServers[k] = v
		}
	}
	trans := transport.NewSubprocessCLITransport(stream, transportOptions)
	if err := trans.Connect(ctx); err != nil {
		return err
	}

	c.transport = trans
	return nil
}

// ReceiveMessages returns a channel that yields all messages from Claude
func (c *Client) ReceiveMessages(ctx context.Context) <-chan MessageResult {
	c.mu.Lock()
	transport := c.transport
	c.mu.Unlock()

	if transport == nil {
		ch := make(chan MessageResult, 1)
		ch <- MessageResult{Error: NewCLIConnectionError("Not connected. Call Connect() first.")}
		close(ch)
		return ch
	}

	out := make(chan MessageResult)
	go func() {
		defer close(out)

		msgChan := transport.ReceiveMessages(ctx)
		for data := range msgChan {
			if data.Err != nil {
				out <- MessageResult{Error: data.Err}
				return
			}

			msg, err := parseMessage(data.Data)
			if err != nil {
				out <- MessageResult{Error: err}
				return
			}

			out <- MessageResult{Message: msg}
		}
	}()

	return out
}

// Query sends a new request in streaming mode
//
// Parameters:
//   - prompt: Either a string message or a MessageStream
//   - sessionID: Session identifier for the conversation
func (c *Client) Query(ctx context.Context, prompt any, sessionID string) error {
	c.mu.Lock()
	transport := c.transport
	c.mu.Unlock()

	if transport == nil {
		return NewCLIConnectionError("Not connected. Call Connect() first.")
	}

	switch p := prompt.(type) {
	case string:
		message := map[string]any{
			"type": "user",
			"message": map[string]any{
				"role":    "user",
				"content": p,
			},
			"parent_tool_use_id": nil,
			"session_id":         sessionID,
		}
		return transport.SendRequest(ctx, []map[string]any{message}, map[string]any{"session_id": sessionID})

	case MessageStream:
		var messages []map[string]any
		for {
			msg, err := p.Next(ctx)
			if err != nil {
				return err
			}
			if msg == nil {
				break // End of stream
			}

			// Ensure session_id is set
			if _, ok := msg["session_id"]; !ok {
				msg["session_id"] = sessionID
			}
			messages = append(messages, msg)
		}

		if len(messages) > 0 {
			return transport.SendRequest(ctx, messages, map[string]any{"session_id": sessionID})
		}
		return nil

	default:
		return &SDKError{message: "prompt must be a string or MessageStream"}
	}
}

// Interrupt sends an interrupt signal (only works with streaming mode)
func (c *Client) Interrupt(ctx context.Context) error {
	c.mu.Lock()
	transport := c.transport
	c.mu.Unlock()

	if transport == nil {
		return NewCLIConnectionError("Not connected. Call Connect() first.")
	}

	return transport.Interrupt(ctx)
}

// ReceiveResponse receives messages from Claude until and including a ResultMessage.
//
// This method yields all messages in sequence and automatically terminates
// after yielding a ResultMessage (which indicates the response is complete).
// It's a convenience method over ReceiveMessages() for single-response workflows.
//
// Stopping Behavior:
// - Yields each message as it's received
// - Terminates immediately after yielding a ResultMessage
// - The ResultMessage IS included in the yielded messages
// - If no ResultMessage is received, the iterator continues indefinitely
//
// Returns:
//   - A channel that yields messages until a ResultMessage is received
//
// Example:
//
//	err := client.Query(ctx, "What's the capital of France?", "default")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	messages := client.ReceiveResponse(ctx)
//	for msg := range messages {
//	    if msg.Error != nil {
//	        log.Fatal(msg.Error)
//	    }
//
//	    switch m := msg.Message.(type) {
//	    case *AssistantMessage:
//	        // Handle assistant response
//	    case *ResultMessage:
//	        fmt.Printf("Cost: $%.4f\n", *m.TotalCostUSD)
//	        // Channel will close after this
//	    }
//	}
func (c *Client) ReceiveResponse(ctx context.Context) <-chan MessageResult {
	out := make(chan MessageResult)
	messages := c.ReceiveMessages(ctx)

	go func() {
		defer close(out)

		for msg := range messages {
			out <- msg

			if msg.Error == nil {
				if _, ok := msg.Message.(*ResultMessage); ok {
					return // Terminate after ResultMessage
				}
			}
		}
	}()

	return out
}

// Disconnect closes the connection to Claude
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.transport != nil {
		err := c.transport.Disconnect()
		c.transport = nil
		return err
	}
	return nil
}

// emptyStream represents an empty message stream for interactive use
type emptyStream struct{}

func (e *emptyStream) Next(ctx context.Context) (map[string]any, error) {
	return nil, nil // Always EOF
}
