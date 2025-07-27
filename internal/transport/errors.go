package transport

import (
	"fmt"
)

// TransportError is the base error type for transport errors.
type TransportError struct {
	message string
}

func (e *TransportError) Error() string {
	return e.message
}

// CLIConnectionError is returned when unable to connect to Claude Code.
type CLIConnectionError struct {
	TransportError
}

// NewCLIConnectionError creates a new CLIConnectionError.
func NewCLIConnectionError(message string) error {
	return &CLIConnectionError{
		TransportError: TransportError{message: message},
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
			TransportError: TransportError{message: fullMessage},
		},
		CLIPath: cliPath,
	}
}

// ProcessError is returned when the CLI process fails.
type ProcessError struct {
	TransportError
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
		TransportError: TransportError{message: fullMessage},
		ExitCode:       exitCode,
		Stderr:         stderr,
	}
}

// CLIJSONDecodeError is returned when unable to decode JSON from CLI output.
type CLIJSONDecodeError struct {
	TransportError
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
		TransportError: TransportError{message: message},
		Line:           line,
		OriginalError:  originalError,
	}
}