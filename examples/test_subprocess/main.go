package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/davlia/claude-code-sdk-go/internal/transport"
)

func main() {
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
	fmt.Println("Connecting...")
	if err := trans.Connect(ctx); err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer func() {
		fmt.Println("Disconnecting...")
		trans.Disconnect()
	}()

	// Receive messages
	fmt.Println("Receiving messages...")
	messages := trans.ReceiveMessages(ctx)
	messageCount := 0
	for msg := range messages {
		messageCount++
		if msg.Err != nil {
			log.Printf("Error in message %d: %v", messageCount, msg.Err)
			break
		}

		fmt.Printf("Message %d: type=%v\n", messageCount, msg.Data["type"])
		
		// Print result if available
		if msg.Data["type"] == "result" {
			fmt.Printf("Result: %v\n", msg.Data["result"])
		}
	}
	
	fmt.Printf("Total messages received: %d\n", messageCount)
	fmt.Println("Done!")
}