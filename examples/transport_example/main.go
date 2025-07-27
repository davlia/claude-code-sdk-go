package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	claude "github.com/davlia/claude-code-sdk-go"
	"github.com/davlia/claude-code-sdk-go/internal/transport"
)

func main() {
	// Example 1: String prompt (non-streaming)
	fmt.Println("=== Example 1: String prompt ===")
	stringExample()

	// Example 2: Streaming prompt
	fmt.Println("\n=== Example 2: Streaming prompt ===")
	streamingExample()

	// Example 3: Interactive streaming
	fmt.Println("\n=== Example 3: Interactive streaming ===")
	interactiveExample()
}

func stringExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create transport with string prompt
	trans := transport.NewSubprocessCLITransport(
		transport.NewStringPromptStream("What is 2+2?"),
		&transport.Options{
			Model: "claude-sonnet-4-20250514",
		},
	)

	// Connect
	if err := trans.Connect(ctx); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer trans.Disconnect()

	// Receive messages
	messages := trans.ReceiveMessages(ctx)
	for msg := range messages {
		if msg.Err != nil {
			log.Fatal("Error:", msg.Err)
		}

		fmt.Printf("Message: %+v\n", msg.Data)
	}
}

func streamingExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a streaming prompt
	prompt := &customStream{
		messages: []map[string]any{
			{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": "Tell me a short joke",
				},
				"parent_tool_use_id": nil,
				"session_id":         "joke-session",
			},
		},
	}

	// Create transport with streaming
	trans := transport.NewSubprocessCLITransport(
		prompt,
		&transport.Options{
			Model: "claude-sonnet-4-20250514",
		},
	).WithStreaming(true).WithCloseStdinAfterPrompt(true)

	// Connect
	if err := trans.Connect(ctx); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer trans.Disconnect()

	// Receive messages
	messages := trans.ReceiveMessages(ctx)
	for msg := range messages {
		if msg.Err != nil {
			log.Fatal("Error:", msg.Err)
		}

		fmt.Printf("Message type: %v\n", msg.Data["type"])
	}
}

func interactiveExample() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create an empty stream for interactive use
	prompt := &emptyStream{}

	// Create transport
	trans := transport.NewSubprocessCLITransport(
		prompt,
		&transport.Options{
			Model: "claude-sonnet-4-20250514",
		},
	).WithStreaming(true).WithSessionID("interactive-session")

	// Connect
	if err := trans.Connect(ctx); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer trans.Disconnect()

	// Send first message
	messages := []map[string]any{
		{
			"type": "user",
			"message": map[string]any{
				"role":    "user",
				"content": "Hello! Can you help me with math?",
			},
			"parent_tool_use_id": nil,
		},
	}

	if err := trans.SendRequest(ctx, messages, map[string]any{"session_id": "interactive-session"}); err != nil {
		log.Fatal("Failed to send request:", err)
	}

	// Start receiving messages in a goroutine
	go func() {
		msgChan := trans.ReceiveMessages(ctx)
		for msg := range msgChan {
			if msg.Err != nil {
				log.Println("Error:", msg.Err)
				return
			}

			fmt.Printf("Received: %v\n", msg.Data["type"])

			// If we get an assistant message, send a follow-up
			if msg.Data["type"] == "assistant" {
				time.Sleep(1 * time.Second)

				followUp := []map[string]any{
					{
						"type": "user",
						"message": map[string]any{
							"role":    "user",
							"content": "What's 15% of 80?",
						},
						"parent_tool_use_id": nil,
					},
				}

				if err := trans.SendRequest(ctx, followUp, map[string]any{"session_id": "interactive-session"}); err != nil {
					log.Println("Failed to send follow-up:", err)
				}
			}
		}
	}()

	// Wait for completion
	<-ctx.Done()
}

// customStream implements MessageStream for testing
type customStream struct {
	messages []map[string]any
	index    int
}

func (cs *customStream) Next(ctx context.Context) (map[string]any, error) {
	if cs.index >= len(cs.messages) {
		return nil, io.EOF
	}
	msg := cs.messages[cs.index]
	cs.index++
	return msg, nil
}

// emptyStream implements MessageStream for interactive use
type emptyStream struct{}

func (es *emptyStream) Next(ctx context.Context) (map[string]any, error) {
	// Block forever - we'll use SendRequest to send messages
	<-ctx.Done()
	return nil, ctx.Err()
}

// streamAdapter adapts claude.MessageStream to transport.MessageStream
type streamAdapter struct {
	stream claude.MessageStream
}

func (sa *streamAdapter) Next(ctx context.Context) (map[string]any, error) {
	return sa.stream.Next(ctx)
}