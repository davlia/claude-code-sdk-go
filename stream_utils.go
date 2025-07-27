package claude

// IsStringPrompt checks if a MessageStream is a simple string prompt.
// This is useful for determining whether to use streaming mode or not.
func IsStringPrompt(stream MessageStream) bool {
	if stream == nil {
		return false
	}
	
	// Check if it's an emptyStream
	if _, ok := stream.(*emptyStream); ok {
		return false
	}
	
	// Check if it's a stringPrompt
	if _, ok := stream.(*stringPrompt); ok {
		return true
	}
	
	return false
}

// NewStringPromptStream creates a MessageStream from a string prompt.
func NewStringPromptStream(prompt string) MessageStream {
	return &stringPrompt{prompt: prompt}
}

// NewEmptyStream creates an empty MessageStream for interactive use.
func NewEmptyStream() MessageStream {
	return &emptyStream{}
}