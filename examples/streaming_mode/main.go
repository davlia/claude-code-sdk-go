package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/davlia/claude-code-sdk-go"
)

// customMessageStream implements MessageStream for custom streaming
type customMessageStream struct {
	messages []map[string]any
	index    int
}

func (s *customMessageStream) Next(ctx context.Context) (map[string]any, error) {
	if s.index >= len(s.messages) {
		return nil, nil // EOF
	}
	msg := s.messages[s.index]
	s.index++
	return msg, nil
}

func main() {
	ctx := context.Background()

	// Create a custom message stream
	stream := &customMessageStream{
		messages: []map[string]any{
			{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": "Hello Claude!",
				},
				"parent_tool_use_id": nil,
				"session_id":         "example-session",
			},
			{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": "How are you today?",
				},
				"parent_tool_use_id": nil,
				"session_id":         "example-session",
			},
			{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": "What's the weather like where you are?",
				},
				"parent_tool_use_id": nil,
				"session_id":         "example-session",
			},
		},
	}

	fmt.Println("=== Streaming Mode Example ===")
	fmt.Println("Sending multiple messages in stream...")
	fmt.Println()

	// Query with the message stream
	messages, err := claude.Query(ctx, stream, nil)
	if err != nil {
		log.Fatal(err)
	}

	// Process all responses
	for msg := range messages {
		if msg.Error != nil {
			log.Fatal(msg.Error)
		}

		switch m := msg.Message.(type) {
		case *claude.UserMessage:
			fmt.Printf("User: %s\n", m.Content)

		case *claude.AssistantMessage:
			fmt.Println("Claude:")
			for _, block := range m.Content {
				if textBlock, ok := block.(*claude.TextBlock); ok {
					fmt.Println(textBlock.Text)
				}
			}
			fmt.Println()

		case *claude.ResultMessage:
			fmt.Printf("Session completed. Total turns: %d\n", m.NumTurns)
			if m.TotalCostUSD != nil {
				fmt.Printf("Total cost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}
}
