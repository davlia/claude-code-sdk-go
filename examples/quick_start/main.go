package main

import (
	"context"
	"fmt"
	"log"

	claude "github.com/davlia/claude-code-sdk-go"
)

func main() {
	ctx := context.Background()

	// Run all examples
	if err := basicExample(ctx); err != nil {
		log.Fatal(err)
	}

	if err := withOptionsExample(ctx); err != nil {
		log.Fatal(err)
	}

	if err := withToolsExample(ctx); err != nil {
		log.Fatal(err)
	}
}

func basicExample(ctx context.Context) error {
	fmt.Println("=== Basic Example ===")

	messages, err := claude.Query(ctx, "What is 2 + 2?", nil)
	if err != nil {
		return err
	}

	for msg := range messages {
		if msg.Error != nil {
			return msg.Error
		}

		if assistantMsg, ok := msg.Message.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(*claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}

	fmt.Println()
	return nil
}

func withOptionsExample(ctx context.Context) error {
	fmt.Println("=== With Options Example ===")

	options := &claude.Options{
		SystemPrompt: "You are a helpful assistant that explains things simply.",
		MaxTurns:     intPtr(1),
	}

	messages, err := claude.Query(ctx, "Explain what Go is in one sentence.", options)
	if err != nil {
		return err
	}

	for msg := range messages {
		if msg.Error != nil {
			return msg.Error
		}

		if assistantMsg, ok := msg.Message.(*claude.AssistantMessage); ok {
			for _, block := range assistantMsg.Content {
				if textBlock, ok := block.(*claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		}
	}

	fmt.Println()
	return nil
}

func withToolsExample(ctx context.Context) error {
	fmt.Println("=== With Tools Example ===")

	options := &claude.Options{
		AllowedTools: []string{"Read", "Write"},
		SystemPrompt: "You are a helpful file assistant.",
	}

	messages, err := claude.Query(ctx, "Create a file called hello.txt with 'Hello, World!' in it", options)
	if err != nil {
		return err
	}

	for msg := range messages {
		if msg.Error != nil {
			return msg.Error
		}

		switch m := msg.Message.(type) {
		case *claude.AssistantMessage:
			for _, block := range m.Content {
				if textBlock, ok := block.(*claude.TextBlock); ok {
					fmt.Printf("Claude: %s\n", textBlock.Text)
				}
			}
		case *claude.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\nCost: $%.4f\n", *m.TotalCostUSD)
			}
		}
	}

	fmt.Println()
	return nil
}

func intPtr(i int) *int {
	return &i
}
