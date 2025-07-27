package claude

import (
	"fmt"
)

// SDKError is the base error type for all Claude SDK errors.
type SDKError struct {
	message string
}

func (e *SDKError) Error() string {
	return e.message
}

// CLIConnectionError is returned when unable to connect to Claude Code.
type CLIConnectionError struct {
	SDKError
}

// NewCLIConnectionError creates a new CLIConnectionError.
func NewCLIConnectionError(message string) error {
	return &CLIConnectionError{
		SDKError: SDKError{message: message},
	}
}

// CLINotFoundError is returned when Claude Code is not found or not installed.
type CLINotFoundError struct {
	CLIConnectionError
	CLIPath string
}

// NewCLINotFoundError creates a new CLINotFoundError.
func NewCLINotFoundError(message string, cliPath string) error {
	fullMessage := message
	if cliPath != "" {
		fullMessage = fmt.Sprintf("%s: %s", message, cliPath)
	}
	return &CLINotFoundError{
		CLIConnectionError: CLIConnectionError{
			SDKError: SDKError{message: fullMessage},
		},
		CLIPath: cliPath,
	}
}

// ProcessError is returned when the CLI process fails.
type ProcessError struct {
	SDKError
	ExitCode int
	Stderr   string
}

// NewProcessError creates a new ProcessError.
func NewProcessError(message string, exitCode int, stderr string) error {
	fullMessage := message
	if exitCode != 0 {
		fullMessage = fmt.Sprintf("%s (exit code: %d)", message, exitCode)
	}
	if stderr != "" {
		fullMessage = fmt.Sprintf("%s\nError output: %s", fullMessage, stderr)
	}
	return &ProcessError{
		SDKError: SDKError{message: fullMessage},
		ExitCode: exitCode,
		Stderr:   stderr,
	}
}

// CLIJSONDecodeError is returned when unable to decode JSON from CLI output.
type CLIJSONDecodeError struct {
	SDKError
	Line          string
	OriginalError error
}

// NewCLIJSONDecodeError creates a new CLIJSONDecodeError.
func NewCLIJSONDecodeError(line string, originalError error) error {
	message := fmt.Sprintf("Failed to decode JSON: %s", line)
	if len(line) > 100 {
		message = fmt.Sprintf("Failed to decode JSON: %s...", line[:100])
	}
	return &CLIJSONDecodeError{
		SDKError:      SDKError{message: message},
		Line:          line,
		OriginalError: originalError,
	}
}

// MessageParseError is returned when unable to parse a message from CLI output.
type MessageParseError struct {
	SDKError
	Data map[string]interface{}
}

// NewMessageParseError creates a new MessageParseError.
func NewMessageParseError(message string, data map[string]interface{}) error {
	return &MessageParseError{
		SDKError: SDKError{message: message},
		Data:     data,
	}
}
