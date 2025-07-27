package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

// cliTransport defines the interface for communication with Claude Code CLI.
// This is an internal interface, not part of the public API.
type cliTransport interface {
	connect(ctx context.Context) error
	disconnect() error
	receiveMessages(ctx context.Context) <-chan messageData
	sendRequest(ctx context.Context, messages []map[string]any, metadata map[string]any) error
	interrupt(ctx context.Context) error
}

// messageData wraps message data or error.
// This is an internal type, not part of the public API.
type messageData struct {
	data map[string]any
	err  error
}

// subprocessCLITransport implements cliTransport using subprocess.
// This is an internal type, not part of the public API.
type subprocessCLITransport struct {
	prompt    MessageStream
	options   *Options
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	outChan   chan messageData
	mu        sync.Mutex
	connected bool
	sessionID string
}

// newSubprocessCLITransport creates a new subprocess CLI transport.
// This is an internal function, not part of the public API.
func newSubprocessCLITransport(prompt MessageStream, options *Options) *subprocessCLITransport {
	return &subprocessCLITransport{
		prompt:    prompt,
		options:   options,
		sessionID: "default",
	}
}

func (t *subprocessCLITransport) connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return NewCLIConnectionError("Already connected")
	}

	// Find Claude Code CLI
	cliPath, err := findCLI()
	if err != nil {
		return err
	}

	// Prepare command
	args := []string{"chat"}
	if t.options != nil {
		optionsJSON, marshalErr := json.Marshal(t.options)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal options: %w", marshalErr)
		}
		args = append(args, "--json-options", string(optionsJSON))
	}

	t.cmd = exec.CommandContext(ctx, cliPath, args...)

	// Set environment
	t.cmd.Env = os.Environ()

	// Setup pipes
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := t.cmd.Start(); err != nil {
		return NewProcessError("Failed to start Claude Code", 0, err.Error())
	}

	t.connected = true
	t.outChan = make(chan messageData, 100)

	// Start reading stdout
	go t.readOutput(ctx)

	// Start sending prompts
	go t.sendPrompts(ctx)

	return nil
}

func (t *subprocessCLITransport) disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	t.connected = false

	// Close stdin to signal we're done
	if t.stdin != nil {
		t.stdin.Close()
	}

	// Wait for process to exit or force kill after timeout
	done := make(chan error, 1)
	go func() {
		done <- t.cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited normally
	case <-context.Background().Done():
		// Force kill if timeout
		if t.cmd.Process != nil {
			_ = t.cmd.Process.Kill()
		}
	}

	// Close output channel
	if t.outChan != nil {
		close(t.outChan)
	}

	return nil
}

func (t *subprocessCLITransport) receiveMessages(ctx context.Context) <-chan messageData {
	return t.outChan
}

func (t *subprocessCLITransport) sendRequest(
	ctx context.Context,
	messages []map[string]any,
	metadata map[string]any,
) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return NewCLIConnectionError("Not connected")
	}

	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		if _, err := fmt.Fprintf(t.stdin, "%s\n", data); err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}
	}

	return nil
}

func (t *subprocessCLITransport) interrupt(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected || t.cmd.Process == nil {
		return NewCLIConnectionError("Not connected")
	}

	// Send SIGINT to the process
	return t.cmd.Process.Signal(syscall.SIGINT)
}

func (t *subprocessCLITransport) readOutput(ctx context.Context) {
	defer func() {
		if t.outChan != nil {
			close(t.outChan)
		}
	}()

	scanner := bufio.NewScanner(t.stdout)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			select {
			case t.outChan <- messageData{err: NewCLIJSONDecodeError(line, err)}:
			case <-ctx.Done():
				return
			}
			continue
		}

		select {
		case t.outChan <- messageData{data: data}:
		case <-ctx.Done():
			return
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case t.outChan <- messageData{err: fmt.Errorf("error reading output: %w", err)}:
		case <-ctx.Done():
		}
	}
}

func (t *subprocessCLITransport) sendPrompts(ctx context.Context) {
	if t.prompt == nil {
		return
	}

	for {
		msg, err := t.prompt.Next(ctx)
		if err != nil {
			select {
			case t.outChan <- messageData{err: err}:
			case <-ctx.Done():
			}
			return
		}

		if msg == nil {
			// End of stream
			return
		}

		// Set session ID if not present
		if _, ok := msg["session_id"]; !ok {
			msg["session_id"] = t.sessionID
		}

		if err := t.sendRequest(ctx, []map[string]any{msg}, nil); err != nil {
			select {
			case t.outChan <- messageData{err: err}:
			case <-ctx.Done():
			}
			return
		}
	}
}

// findCLI locates the Claude Code CLI executable.
func findCLI() (string, error) {
	// Check CLAUDE_CODE_CLI_PATH environment variable
	if path := os.Getenv("CLAUDE_CODE_CLI_PATH"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", NewCLINotFoundError("Claude Code not found at CLAUDE_CODE_CLI_PATH", path)
	}

	// Check common installation paths
	paths := []string{
		"claude-code",
		"npx claude-code",
	}

	// Add npm global bin to paths
	if npmBin := getNpmGlobalBin(); npmBin != "" {
		paths = append(paths, filepath.Join(npmBin, "claude-code"))
	}

	// Try to find in PATH
	for _, p := range paths {
		if path, err := exec.LookPath(p); err == nil {
			return path, nil
		}
	}

	// Try with full npx command
	if path, err := exec.LookPath("npx"); err == nil {
		// Test if claude-code is available via npx
		cmd := exec.Command(path, "claude-code", "--version")
		if err := cmd.Run(); err == nil {
			return "npx", nil
		}
	}

	return "", NewCLINotFoundError(
		"Claude Code not found. Please install with: npm install -g @anthropic-ai/claude-code",
		"",
	)
}

// getNpmGlobalBin returns the npm global bin directory.
func getNpmGlobalBin() string {
	cmd := exec.Command("npm", "bin", "-g")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
