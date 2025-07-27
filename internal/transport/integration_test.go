// +build integration

package transport_test

import (
	"context"
	"testing"
	"time"

	claude "github.com/davlia/claude-code-sdk-go"
	"github.com/davlia/claude-code-sdk-go/internal/transport"
)

func TestIntegration_FullConversation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Setup options with various configurations
	options := claude.NewOptions()
	options.SystemPrompt = "You are a helpful math tutor"
	options.Model = "claude-3-sonnet"
	options.PermissionMode = claude.PermissionModeDefault
	
	// Create an interactive stream
	prompt := claude.NewEmptyStream()
	
	trans := transport.NewSubprocessCLITransport(prompt, options).
		WithStreaming(true).
		WithSessionID("math-session")

	// Connect
	if err := trans.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer trans.Disconnect()

	// Conversation flow
	questions := []string{
		"Hello! Can you help me learn about fractions?",
		"What is 1/2 + 1/4?",
		"Can you explain why that's the answer?",
	}

	// Start receiving messages
	msgChan := trans.ReceiveMessages(ctx)
	go func() {
		for msg := range msgChan {
			if msg.Err != nil {
				t.Logf("Error: %v", msg.Err)
				return
			}
			
			msgType, _ := msg.Data["type"].(string)
			t.Logf("Received: %s", msgType)
			
			if msgType == "assistant" {
				if content, ok := msg.Data["content"].([]interface{}); ok {
					for _, block := range content {
						if textBlock, ok := block.(map[string]interface{}); ok {
							if text, ok := textBlock["text"].(string); ok {
								t.Logf("Assistant: %s", text)
							}
						}
					}
				}
			}
		}
	}()

	// Send questions with delays
	for i, question := range questions {
		t.Logf("Sending question %d: %s", i+1, question)
		
		msg := []map[string]any{
			{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": question,
				},
				"parent_tool_use_id": nil,
			},
		}
		
		if err := trans.SendRequest(ctx, msg, map[string]any{"session_id": "math-session"}); err != nil {
			t.Errorf("Failed to send question %d: %v", i+1, err)
			break
		}
		
		// Wait between questions
		time.Sleep(3 * time.Second)
	}

	// Let the conversation finish
	time.Sleep(5 * time.Second)
}