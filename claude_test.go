package claude

import (
	"context"
	"testing"
)

func TestNewOptions(t *testing.T) {
	opts := NewOptions()

	if opts.MaxThinkingTokens != 8000 {
		t.Errorf("Expected MaxThinkingTokens to be 8000, got %d", opts.MaxThinkingTokens)
	}

	if len(opts.AllowedTools) != 0 {
		t.Errorf("Expected AllowedTools to be empty, got %v", opts.AllowedTools)
	}

	if opts.MCPServers == nil {
		t.Error("Expected MCPServers to be initialized")
	}
}

func TestPermissionModeValues(t *testing.T) {
	tests := []struct {
		mode     PermissionMode
		expected string
	}{
		{PermissionModeDefault, "default"},
		{PermissionModeAcceptEdits, "acceptEdits"},
		{PermissionModeBypassPermissions, "bypassPermissions"},
	}

	for _, test := range tests {
		if string(test.mode) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, string(test.mode))
		}
	}
}

func TestMCPServerTypes(t *testing.T) {
	stdioConfig := MCPStdioServerConfig{
		Command: "test-command",
		Args:    []string{"arg1", "arg2"},
	}

	if stdioConfig.GetType() != MCPServerTypeStdio {
		t.Errorf("Expected stdio type, got %s", stdioConfig.GetType())
	}

	sseConfig := MCPSSEServerConfig{
		Type: MCPServerTypeSSE,
		URL:  "http://example.com",
	}

	if sseConfig.GetType() != MCPServerTypeSSE {
		t.Errorf("Expected sse type, got %s", sseConfig.GetType())
	}
}

func TestStringPrompt(t *testing.T) {
	prompt := &stringPrompt{prompt: "Hello, Claude!"}
	ctx := context.Background()

	// First call should return the message
	msg, err := prompt.Next(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if msg == nil {
		t.Fatal("Expected message, got nil")
	}

	content, ok := msg["message"].(map[string]any)
	if !ok {
		t.Fatal("Expected message field to be a map")
	}

	if content["content"] != "Hello, Claude!" {
		t.Errorf("Expected content 'Hello, Claude!', got %v", content["content"])
	}

	// Second call should return EOF
	msg2, err2 := prompt.Next(ctx)
	if err2 != nil {
		t.Fatalf("Unexpected error on second call: %v", err2)
	}

	if msg2 != nil {
		t.Error("Expected nil on second call, got message")
	}
}

func TestEmptyStream(t *testing.T) {
	stream := &emptyStream{}
	ctx := context.Background()

	msg, err := stream.Next(ctx)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if msg != nil {
		t.Error("Expected nil from empty stream")
	}
}

func TestParseTextBlock(t *testing.T) {
	data := map[string]any{
		"type": "text",
		"text": "Hello, world!",
	}

	block, err := parseContentBlock(data)
	if err != nil {
		t.Fatalf("Failed to parse text block: %v", err)
	}

	textBlock, ok := block.(*TextBlock)
	if !ok {
		t.Fatal("Expected TextBlock type")
	}

	if textBlock.Text != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got %s", textBlock.Text)
	}
}

func TestParseToolUseBlock(t *testing.T) {
	data := map[string]any{
		"type": "tool_use",
		"id":   "tool-123",
		"name": "Read",
		"input": map[string]any{
			"file": "test.txt",
		},
	}

	block, err := parseContentBlock(data)
	if err != nil {
		t.Fatalf("Failed to parse tool use block: %v", err)
	}

	toolBlock, ok := block.(*ToolUseBlock)
	if !ok {
		t.Fatal("Expected ToolUseBlock type")
	}

	if toolBlock.ID != "tool-123" {
		t.Errorf("Expected ID 'tool-123', got %s", toolBlock.ID)
	}

	if toolBlock.Name != "Read" {
		t.Errorf("Expected name 'Read', got %s", toolBlock.Name)
	}
}

func TestParseAssistantMessage(t *testing.T) {
	data := map[string]any{
		"type": "assistant",
		"content": []any{
			map[string]any{
				"type": "text",
				"text": "Hello!",
			},
			map[string]any{
				"type": "tool_use",
				"id":   "tool-456",
				"name": "Write",
				"input": map[string]any{
					"file":    "output.txt",
					"content": "Test content",
				},
			},
		},
	}

	msg, err := parseMessage(data)
	if err != nil {
		t.Fatalf("Failed to parse assistant message: %v", err)
	}

	assistantMsg, ok := msg.(*AssistantMessage)
	if !ok {
		t.Fatal("Expected AssistantMessage type")
	}

	if len(assistantMsg.Content) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(assistantMsg.Content))
	}

	// Check first block is text
	if _, ok := assistantMsg.Content[0].(*TextBlock); !ok {
		t.Error("Expected first block to be TextBlock")
	}

	// Check second block is tool use
	if _, ok := assistantMsg.Content[1].(*ToolUseBlock); !ok {
		t.Error("Expected second block to be ToolUseBlock")
	}
}

func TestParseResultMessage(t *testing.T) {
	costValue := 0.0025
	data := map[string]any{
		"type":            "result",
		"subtype":         "conversation_end",
		"duration_ms":     float64(1500),
		"duration_api_ms": float64(1200),
		"is_error":        false,
		"num_turns":       float64(3),
		"session_id":      "test-session",
		"total_cost_usd":  costValue,
	}

	msg, err := parseMessage(data)
	if err != nil {
		t.Fatalf("Failed to parse result message: %v", err)
	}

	resultMsg, ok := msg.(*ResultMessage)
	if !ok {
		t.Fatal("Expected ResultMessage type")
	}

	if resultMsg.DurationMS != 1500 {
		t.Errorf("Expected DurationMS 1500, got %d", resultMsg.DurationMS)
	}

	if resultMsg.NumTurns != 3 {
		t.Errorf("Expected NumTurns 3, got %d", resultMsg.NumTurns)
	}

	if resultMsg.TotalCostUSD == nil || *resultMsg.TotalCostUSD != costValue {
		t.Errorf("Expected TotalCostUSD %f, got %v", costValue, resultMsg.TotalCostUSD)
	}
}

func TestClientConnectionLifecycle(t *testing.T) {
	client := NewClient(nil)

	// Test double connect
	if client.transport != nil {
		t.Error("Expected transport to be nil before connection")
	}

	// Note: We can't test actual connection without mocking the CLI
	// This is more of a structure/API test
}

func TestMessageResultChannel(t *testing.T) {
	// Test channel behavior
	ch := make(chan MessageResult, 2)

	// Send a message
	ch <- MessageResult{
		Message: &UserMessage{Content: "Test"},
		Error:   nil,
	}

	// Send an error
	ch <- MessageResult{
		Message: nil,
		Error:   NewCLIConnectionError("Test error"),
	}

	close(ch)

	// Read first result
	result1 := <-ch
	if result1.Error != nil {
		t.Error("Expected first result to have no error")
	}

	if _, ok := result1.Message.(*UserMessage); !ok {
		t.Error("Expected first result to have UserMessage")
	}

	// Read second result
	result2 := <-ch
	if result2.Error == nil {
		t.Error("Expected second result to have error")
	}

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Error("Expected channel to be closed")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a message stream that blocks
	stream := &blockingStream{
		blockCh: make(chan struct{}),
	}

	// Cancel context immediately
	cancel()

	// Next should respect context cancellation
	_, err := stream.Next(ctx)
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
}

// Helper types for testing

type blockingStream struct {
	blockCh chan struct{}
}

func (s *blockingStream) Next(ctx context.Context) (map[string]any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.blockCh:
		return nil, nil
	}
}

func TestErrorTypes(t *testing.T) {
	// Test CLINotFoundError
	err1 := NewCLINotFoundError("Claude Code not found", "/usr/local/bin/claude-code")
	expected := "Claude Code not found: /usr/local/bin/claude-code"
	if err1.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err1.Error())
	}

	// Test ProcessError
	err2 := NewProcessError("Command failed", 1, "stderr output")
	processErr, ok := err2.(*ProcessError)
	if !ok {
		t.Fatal("Expected ProcessError type")
	}
	if processErr.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", processErr.ExitCode)
	}

	// Test CLIJSONDecodeError
	origErr := &syntaxError{msg: "invalid json"}
	longLine := "invalid json line that is very long and should be truncated after 100 characters " +
		"to avoid massive error messages"
	err3 := NewCLIJSONDecodeError(longLine, origErr)
	if len(err3.Error()) > 200 {
		t.Error("Expected error message to be truncated")
	}
}

type syntaxError struct {
	msg string
}

func (e *syntaxError) Error() string {
	return e.msg
}

// Benchmark for message parsing
func BenchmarkParseAssistantMessage(b *testing.B) {
	data := map[string]any{
		"type": "assistant",
		"content": []any{
			map[string]any{
				"type": "text",
				"text": "This is a longer text message to simulate real usage.",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parseMessage(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Example usage for documentation
func ExampleQuery() {
	ctx := context.Background()

	// Simple query
	messages, err := Query(ctx, "What is 2 + 2?", nil)
	if err != nil {
		// Handle error
		return
	}

	for msg := range messages {
		if msg.Error != nil {
			// Handle error
			continue
		}
		// Process message
		_ = msg.Message
	}
}

func ExampleNewClient() {
	ctx := context.Background()

	// Create interactive client
	client := NewClient(nil)

	// Connect
	err := client.Connect(ctx, nil)
	if err != nil {
		// Handle error
		return
	}
	defer client.Disconnect()

	// Send message
	err = client.Query(ctx, "Hello, Claude!", "default")
	if err != nil {
		// Handle error
		return
	}

	// Receive response
	messages := client.ReceiveResponse(ctx)
	for msg := range messages {
		// Process messages
		_ = msg
	}
}
