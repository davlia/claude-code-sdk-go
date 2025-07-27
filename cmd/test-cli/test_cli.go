package main

import (
	"fmt"
	"os/exec"
)

func main() {
	// Test if claude-code CLI works
	cmd := exec.Command("claude-code", "--version")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error running claude-code: %v\n", err)
		return
	}
	fmt.Printf("Claude Code version: %s\n", output)
	
	// Test simple prompt
	cmd2 := exec.Command("claude-code", "-p", "What is 2+2?")
	output2, err2 := cmd2.Output()
	if err2 != nil {
		fmt.Printf("Error running claude-code with prompt: %v\n", err2)
		return
	}
	fmt.Printf("Response: %s\n", output2)
}