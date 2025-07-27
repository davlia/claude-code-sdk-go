package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	claude "github.com/davlia/claude-code-sdk-go"
)

func main() {
	ctx := context.Background()

	// Create client with options
	options := &claude.Options{
		SystemPrompt: "You are a helpful assistant in an interactive session.",
	}
	client := claude.NewClient(options)

	// Connect with empty stream for interactive use
	fmt.Println("Connecting to Claude...")
	if err := client.Connect(ctx, nil); err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	fmt.Println("Connected! Type your messages (or 'quit' to exit):")
	fmt.Println()

	// Start a goroutine to handle incoming messages
	go handleMessages(ctx, client)

	// Read user input and send messages
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "quit" || input == "exit" {
			break
		}

		if input == "" {
			continue
		}

		// Send message to Claude
		if err := client.Query(ctx, input, "default"); err != nil {
			fmt.Printf("Error sending message: %v\n", err)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input: %v", err)
	}

	fmt.Println("\nGoodbye!")
}

func handleMessages(ctx context.Context, client *claude.Client) {
	messages := client.ReceiveMessages(ctx)

	for msg := range messages {
		if msg.Error != nil {
			fmt.Printf("\nError: %v\n", msg.Error)
			fmt.Print("> ")
			continue
		}

		switch m := msg.Message.(type) {
		case *claude.AssistantMessage:
			fmt.Println("\nClaude:")
			for _, block := range m.Content {
				switch b := block.(type) {
				case *claude.TextBlock:
					fmt.Println(b.Text)
				case *claude.ToolUseBlock:
					fmt.Printf("[Using tool: %s]\n", b.Name)
				}
			}
			fmt.Print("\n> ")

		case *claude.SystemMessage:
			// Optionally display system messages
			if m.Subtype == "usage" || m.Subtype == "info" {
				fmt.Printf("\n[System: %s]\n", m.Subtype)
				fmt.Print("> ")
			}

		case *claude.ResultMessage:
			if m.TotalCostUSD != nil && *m.TotalCostUSD > 0 {
				fmt.Printf("\n[Session cost: $%.4f]\n", *m.TotalCostUSD)
				fmt.Print("> ")
			}
		}
	}
}
