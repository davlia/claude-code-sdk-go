package transport_test

import (
	"context"
	"os"
	"testing"
	"time"

	claude "github.com/davlia/claude-code-sdk-go"
	"github.com/davlia/claude-code-sdk-go/internal/transport"
)

func TestSubprocessCLITransport_StringPrompt(t *testing.T) {
	if os.Getenv("CLAUDE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set CLAUDE_INTEGRATION_TEST=1 to run.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := claude.NewStringPromptStream("What is 2+2?")
	options := claude.NewOptions()

	trans := transport.NewSubprocessCLITransport(prompt, options)

	if err := trans.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer trans.Disconnect()

	if !trans.IsConnected() {
		t.Error("Transport should be connected")
	}

	messages := trans.ReceiveMessages(ctx)
	messageCount := 0
	
	for msg := range messages {
		if msg.Err != nil {
			t.Errorf("Received error: %v", msg.Err)
			continue
		}
		messageCount++
		
		if msgType, ok := msg.Data["type"].(string); ok {
			t.Logf("Received message type: %s", msgType)
		}
	}

	if messageCount == 0 {
		t.Error("No messages received")
	}
}

func TestSubprocessCLITransport_StreamingPrompt(t *testing.T) {
	if os.Getenv("CLAUDE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set CLAUDE_INTEGRATION_TEST=1 to run.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a custom stream
	prompt := &testStream{
		messages: []map[string]any{
			{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": "Say 'hello'",
				},
				"parent_tool_use_id": nil,
				"session_id":         "test-session",
			},
		},
	}

	options := claude.NewOptions()
	trans := transport.NewSubprocessCLITransport(prompt, options).
		WithStreaming(true).
		WithCloseStdinAfterPrompt(true)

	if err := trans.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer trans.Disconnect()

	messages := trans.ReceiveMessages(ctx)
	messageCount := 0
	
	for msg := range messages {
		if msg.Err != nil {
			t.Errorf("Received error: %v", msg.Err)
			continue
		}
		messageCount++
	}

	if messageCount == 0 {
		t.Error("No messages received")
	}
}

func TestSubprocessCLITransport_Interactive(t *testing.T) {
	if os.Getenv("CLAUDE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set CLAUDE_INTEGRATION_TEST=1 to run.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := claude.NewEmptyStream()
	options := claude.NewOptions()

	trans := transport.NewSubprocessCLITransport(prompt, options).
		WithStreaming(true).
		WithSessionID("test-interactive")

	if err := trans.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer trans.Disconnect()

	// Send a message
	messages := []map[string]any{
		{
			"type": "user",
			"message": map[string]any{
				"role":    "user",
				"content": "Hello",
			},
			"parent_tool_use_id": nil,
		},
	}

	if err := trans.SendRequest(ctx, messages, map[string]any{"session_id": "test-interactive"}); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Receive some messages
	msgChan := trans.ReceiveMessages(ctx)
	receivedAssistant := false
	
	timeout := time.After(10 * time.Second)
	for {
		select {
		case msg := <-msgChan:
			if msg.Err != nil {
				t.Errorf("Received error: %v", msg.Err)
				return
			}
			
			if msgType, ok := msg.Data["type"].(string); ok && msgType == "assistant" {
				receivedAssistant = true
				// Try to interrupt
				if err := trans.Interrupt(ctx); err == nil {
					t.Log("Successfully sent interrupt")
				}
				return
			}
		case <-timeout:
			t.Error("Timeout waiting for assistant message")
			return
		}
	}

	if !receivedAssistant {
		t.Error("Did not receive assistant message")
	}
}

func TestSubprocessCLITransport_CLINotFound(t *testing.T) {
	prompt := claude.NewStringPromptStream("test")
	options := claude.NewOptions()

	trans := transport.NewSubprocessCLITransport(prompt, options).
		WithCLIPath("/nonexistent/path/to/claude")

	ctx := context.Background()
	err := trans.Connect(ctx)
	
	if err == nil {
		trans.Disconnect()
		t.Error("Expected error when CLI not found")
	}
}

func TestSubprocessCLITransport_InvalidWorkingDir(t *testing.T) {
	prompt := claude.NewStringPromptStream("test")
	options := claude.NewOptions()
	options.Cwd = "/nonexistent/directory"

	trans := transport.NewSubprocessCLITransport(prompt, options)

	ctx := context.Background()
	err := trans.Connect(ctx)
	
	if err == nil {
		trans.Disconnect()
		t.Error("Expected error when working directory doesn't exist")
	}
}

func TestSubprocessCLITransport_Options(t *testing.T) {
	prompt := claude.NewStringPromptStream("test")
	
	options := claude.NewOptions()
	options.SystemPrompt = "You are a helpful assistant"
	options.Model = "claude-3-sonnet"
	options.AllowedTools = []string{"read", "write"}
	options.DisallowedTools = []string{"execute"}
	maxTurns := 5
	options.MaxTurns = &maxTurns
	options.PermissionMode = claude.PermissionModeAcceptEdits

	trans := transport.NewSubprocessCLITransport(prompt, options)
	
	// Just verify it creates without error
	if trans == nil {
		t.Error("Failed to create transport with options")
	}
}

// testStream implements MessageStream for testing
type testStream struct {
	messages []map[string]any
	index    int
}

func (s *testStream) Next(ctx context.Context) (map[string]any, error) {
	if s.index >= len(s.messages) {
		return nil, nil // EOF
	}
	msg := s.messages[s.index]
	s.index++
	return msg, nil
}