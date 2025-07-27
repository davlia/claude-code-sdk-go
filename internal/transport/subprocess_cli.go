package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	maxBufferSize  = 1024 * 1024     // 1MB buffer limit
	maxStderrSize  = 10 * 1024 * 1024 // 10MB stderr limit
	stderrTimeout  = 30 * time.Second
	disconnectTimeout = 5 * time.Second
)

// SubprocessCLITransport implements Transport interface using subprocess.
type SubprocessCLITransport struct {
	prompt               MessageStream
	options              *Options
	cliPath              string
	closeStdinAfterPrompt bool
	
	// Process management
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	stdinChan     chan []byte
	outChan       chan MessageData
	
	// Control request handling
	pendingControlResponses map[string]map[string]any
	requestCounter         uint64
	
	// Connection state
	mu            sync.RWMutex
	connected     bool
	isStreaming   bool
	sessionID     string
	taskGroup     sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

// MessageData wraps message data or error.
type MessageData struct {
	Data map[string]any
	Err  error
}

// safeSend safely sends data to the output channel
func (t *SubprocessCLITransport) safeSend(msg MessageData) bool {
	t.mu.RLock()
	outChan := t.outChan
	t.mu.RUnlock()
	
	if outChan == nil {
		return false
	}
	
	select {
	case outChan <- msg:
		return true
	case <-t.ctx.Done():
		return false
	default:
		// Channel might be full, try with context
		select {
		case outChan <- msg:
			return true
		case <-t.ctx.Done():
			return false
		}
	}
}

// NewSubprocessCLITransport creates a new subprocess CLI transport.
func NewSubprocessCLITransport(prompt MessageStream, options *Options) *SubprocessCLITransport {
	if options == nil {
		options = NewOptions()
	}
	
	// Determine if this is a streaming prompt
	isStreaming := !IsStringPrompt(prompt)
	
	return &SubprocessCLITransport{
		prompt:                  prompt,
		options:                 options,
		sessionID:               "default",
		isStreaming:             isStreaming,
		pendingControlResponses: make(map[string]map[string]any),
	}
}

// WithSessionID sets the session ID for the transport.
func (t *SubprocessCLITransport) WithSessionID(sessionID string) *SubprocessCLITransport {
	t.sessionID = sessionID
	return t
}

// WithCLIPath sets a custom CLI path.
func (t *SubprocessCLITransport) WithCLIPath(path string) *SubprocessCLITransport {
	t.cliPath = path
	return t
}

// WithCloseStdinAfterPrompt sets whether to close stdin after sending prompt.
func (t *SubprocessCLITransport) WithCloseStdinAfterPrompt(close bool) *SubprocessCLITransport {
	t.closeStdinAfterPrompt = close
	return t
}

// WithStreaming explicitly sets streaming mode.
func (t *SubprocessCLITransport) WithStreaming(streaming bool) *SubprocessCLITransport {
	t.isStreaming = streaming
	return t
}

// Connect starts the subprocess.
func (t *SubprocessCLITransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.connected {
		return NewCLIConnectionError("Already connected")
	}

	// Find CLI if not specified
	if t.cliPath == "" {
		cliPath, err := t.findCLI()
		if err != nil {
			return err
		}
		t.cliPath = cliPath
	}

	// Create context for this connection
	t.ctx, t.cancel = context.WithCancel(ctx)

	// Build command
	args := t.buildCommand()
	t.cmd = exec.CommandContext(t.ctx, t.cliPath, args...)

	// Set environment
	env := os.Environ()
	env = append(env, "CLAUDE_CODE_ENTRYPOINT=sdk-go")
	t.cmd.Env = env

	// Set working directory if specified
	if t.options.Cwd != "" {
		t.cmd.Dir = t.options.Cwd
	}

	// Setup pipes
	var err error
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
		// Check if error is due to working directory
		if t.options.Cwd != "" {
			if _, statErr := os.Stat(t.options.Cwd); os.IsNotExist(statErr) {
				return NewCLIConnectionError(fmt.Sprintf("Working directory does not exist: %s", t.options.Cwd))
			}
		}
		// Check if error is due to CLI not found
		if _, statErr := os.Stat(t.cliPath); os.IsNotExist(statErr) {
			return NewCLINotFoundError(fmt.Sprintf("Claude Code not found at: %s", t.cliPath), t.cliPath)
		}
		return NewProcessError("Failed to start Claude Code", 0, err.Error())
	}

	t.connected = true
	t.outChan = make(chan MessageData, 100)

	// Handle stdin based on mode
	if t.isStreaming {
		t.stdinChan = make(chan []byte, 100)
		t.taskGroup.Add(1)
		go t.handleStdin()
		
		// Start streaming prompts
		t.taskGroup.Add(1)
		go t.streamToStdin()
	} else {
		// String mode: close stdin immediately (backward compatible)
		t.stdin.Close()
		t.stdin = nil
	}

	// Start reading stdout
	t.taskGroup.Add(1)
	go t.readOutput()

	// Start reading stderr
	t.taskGroup.Add(1)
	go t.readStderr()

	// Start a goroutine to coordinate process exit and channel closing
	go func() {
		// Wait for process to exit
		if t.cmd != nil {
			t.cmd.Wait()
		}
		
		// Wait for all reading goroutines to finish
		t.taskGroup.Wait()
		
		// Close the output channel after everything is done
		t.mu.Lock()
		if t.outChan != nil {
			close(t.outChan)
			t.outChan = nil
		}
		t.mu.Unlock()
	}()

	return nil
}

// Disconnect terminates the subprocess.
func (t *SubprocessCLITransport) Disconnect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.connected {
		return nil
	}

	t.connected = false
	
	// Cancel context to signal all goroutines
	if t.cancel != nil {
		t.cancel()
	}

	// Close stdin channel if streaming
	if t.stdinChan != nil {
		close(t.stdinChan)
	}

	// Close stdin to signal we're done
	if t.stdin != nil {
		t.stdin.Close()
	}

	// Wait for process to exit or force kill after timeout
	done := make(chan error, 1)
	go func() {
		if t.cmd != nil {
			done <- t.cmd.Wait()
		} else {
			done <- nil
		}
	}()

	select {
	case <-done:
		// Process exited normally
	case <-time.After(disconnectTimeout):
		// Force kill if timeout
		if t.cmd != nil && t.cmd.Process != nil {
			_ = t.cmd.Process.Kill()
			<-done
		}
	}

	// Wait for all goroutines to finish
	// This includes the goroutine that closes outChan
	t.taskGroup.Wait()

	return nil
}

// ReceiveMessages returns a channel that yields messages.
func (t *SubprocessCLITransport) ReceiveMessages(ctx context.Context) <-chan MessageData {
	return t.outChan
}

// SendRequest sends additional messages in streaming mode.
func (t *SubprocessCLITransport) SendRequest(ctx context.Context, messages []map[string]any, metadata map[string]any) error {
	if !t.isStreaming {
		return NewCLIConnectionError("SendRequest only works in streaming mode")
	}

	t.mu.RLock()
	connected := t.connected
	t.mu.RUnlock()

	if !connected {
		return NewCLIConnectionError("Not connected")
	}

	sessionID := "default"
	if sid, ok := metadata["session_id"].(string); ok {
		sessionID = sid
	}

	for _, msg := range messages {
		// Ensure message has required structure
		if _, hasType := msg["type"]; !hasType {
			msg = map[string]any{
				"type": "user",
				"message": map[string]any{
					"role":    "user",
					"content": msg,
				},
				"parent_tool_use_id": nil,
				"session_id":         sessionID,
			}
		}

		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal message: %w", err)
		}

		select {
		case t.stdinChan <- append(data, '\n'):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// Interrupt sends an interrupt control request.
func (t *SubprocessCLITransport) Interrupt(ctx context.Context) error {
	if !t.isStreaming {
		// For non-streaming mode, send SIGINT to process
		t.mu.RLock()
		cmd := t.cmd
		t.mu.RUnlock()

		if cmd == nil || cmd.Process == nil {
			return NewCLIConnectionError("Not connected")
		}
		
		return cmd.Process.Signal(syscall.SIGINT)
	}

	// For streaming mode, send control request
	request := map[string]any{"subtype": "interrupt"}
	_, err := t.sendControlRequest(ctx, request)
	return err
}

// IsConnected checks if subprocess is running.
func (t *SubprocessCLITransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return t.connected && t.cmd != nil && t.cmd.Process != nil
}

// buildCommand builds CLI command with arguments.
func (t *SubprocessCLITransport) buildCommand() []string {
	cmd := []string{"--output-format", "stream-json", "--verbose"}

	if t.options.SystemPrompt != "" {
		cmd = append(cmd, "--system-prompt", t.options.SystemPrompt)
	}

	if t.options.AppendSystemPrompt != "" {
		cmd = append(cmd, "--append-system-prompt", t.options.AppendSystemPrompt)
	}

	if len(t.options.AllowedTools) > 0 {
		cmd = append(cmd, "--allowedTools", strings.Join(t.options.AllowedTools, ","))
	}

	if t.options.MaxTurns != nil {
		cmd = append(cmd, "--max-turns", fmt.Sprintf("%d", *t.options.MaxTurns))
	}

	if len(t.options.DisallowedTools) > 0 {
		cmd = append(cmd, "--disallowedTools", strings.Join(t.options.DisallowedTools, ","))
	}

	if t.options.Model != "" {
		cmd = append(cmd, "--model", t.options.Model)
	}

	if t.options.PermissionPromptToolName != "" {
		cmd = append(cmd, "--permission-prompt-tool", t.options.PermissionPromptToolName)
	}

	if t.options.PermissionMode != "" {
		cmd = append(cmd, "--permission-mode", string(t.options.PermissionMode))
	}

	if t.options.ContinueConversation {
		cmd = append(cmd, "--continue")
	}

	if t.options.Resume != "" {
		cmd = append(cmd, "--resume", t.options.Resume)
	}

	if len(t.options.MCPServers) > 0 {
		mcpConfig := map[string]any{"mcpServers": t.options.MCPServers}
		configJSON, _ := json.Marshal(mcpConfig)
		cmd = append(cmd, "--mcp-config", string(configJSON))
	}

	// Add prompt handling based on mode
	if t.isStreaming {
		// Streaming mode: use --input-format stream-json
		cmd = append(cmd, "--input-format", "stream-json")
	} else {
		// String mode: use --print with the prompt
		// Try to extract string content from first message
		if t.prompt != nil {
			tempCtx, tempCancel := context.WithCancel(context.Background())
			
			if msg, err := t.prompt.Next(tempCtx); err == nil && msg != nil {
				if message, ok := msg["message"].(map[string]any); ok {
					if content, ok := message["content"].(string); ok {
						cmd = append(cmd, "--print", content)
					}
				}
			}
			tempCancel()
		}
	}

	return cmd
}

// findCLI locates the Claude Code CLI executable.
func (t *SubprocessCLITransport) findCLI() (string, error) {
	// Check CLAUDE_CODE_CLI_PATH environment variable
	if path := os.Getenv("CLAUDE_CODE_CLI_PATH"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		return "", NewCLINotFoundError("Claude Code not found at CLAUDE_CODE_CLI_PATH", path)
	}

	// Try to find in PATH first
	if path, err := exec.LookPath("claude"); err == nil {
		return path, nil
	}

	// Common installation paths
	home, _ := os.UserHomeDir()
	locations := []string{
		filepath.Join(home, ".npm-global", "bin", "claude"),
		"/usr/local/bin/claude",
		filepath.Join(home, ".local", "bin", "claude"),
		filepath.Join(home, "node_modules", ".bin", "claude"),
		filepath.Join(home, ".yarn", "bin", "claude"),
	}

	// Add Windows-specific paths
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			locations = append(locations, 
				filepath.Join(appData, "npm", "claude"),
				filepath.Join(appData, "npm", "cmd"),
			)
		}
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Check if Node.js is installed
	if _, err := exec.LookPath("node"); err != nil {
		return "", NewCLINotFoundError(
			"Claude Code requires Node.js, which is not installed.\n\n"+
			"Install Node.js from: https://nodejs.org/\n"+
			"\nAfter installing Node.js, install Claude Code:\n"+
			"  npm install -g @anthropic-ai/claude-code",
			"",
		)
	}

	return "", NewCLINotFoundError(
		"Claude Code not found. Install with:\n"+
		"  npm install -g @anthropic-ai/claude-code\n"+
		"\nIf already installed locally, try:\n"+
		`  export PATH="$HOME/node_modules/.bin:$PATH"`+"\n"+
		"\nOr specify the path when creating transport:\n"+
		"  transport.WithCLIPath('/path/to/claude')",
		"",
	)
}

// handleStdin manages writing to stdin in streaming mode.
func (t *SubprocessCLITransport) handleStdin() {
	defer t.taskGroup.Done()

	for data := range t.stdinChan {
		if t.stdin == nil {
			break
		}
		
		if _, err := t.stdin.Write(data); err != nil {
			t.safeSend(MessageData{Err: fmt.Errorf("failed to write to stdin: %w", err), Data: nil})
			break
		}
	}
}

// streamToStdin streams messages to stdin for streaming mode.
func (t *SubprocessCLITransport) streamToStdin() {
	defer t.taskGroup.Done()

	if t.prompt == nil {
		return
	}

	for {
		msg, err := t.prompt.Next(t.ctx)
		if err != nil {
			if err != context.Canceled {
				t.safeSend(MessageData{Data: nil, Err: err})
			}
			return
		}

		if msg == nil {
			// End of stream
			break
		}

		// Set session ID if not present
		if _, ok := msg["session_id"]; !ok {
			msg["session_id"] = t.sessionID
		}

		data, err := json.Marshal(msg)
		if err != nil {
			t.safeSend(MessageData{Data: nil, Err: fmt.Errorf("failed to marshal prompt message: %w", err)})
			return
		}

		select {
		case t.stdinChan <- append(data, '\n'):
		case <-t.ctx.Done():
			return
		}
	}

	// Close stdin after prompt if requested
	if t.closeStdinAfterPrompt {
		t.mu.Lock()
		if t.stdin != nil {
			t.stdin.Close()
			t.stdin = nil
		}
		t.mu.Unlock()
	}
}

// readOutput reads and processes stdout.
func (t *SubprocessCLITransport) readOutput() {
	defer t.taskGroup.Done()

	scanner := bufio.NewScanner(t.stdout)
	scanner.Buffer(make([]byte, maxBufferSize), maxBufferSize)

	jsonBuffer := ""

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Handle multiple JSON objects on same line
		jsonLines := strings.Split(line, "\n")
		
		for _, jsonLine := range jsonLines {
			jsonLine = strings.TrimSpace(jsonLine)
			if jsonLine == "" {
				continue
			}

			// Accumulate JSON until we can parse it
			jsonBuffer += jsonLine

			if len(jsonBuffer) > maxBufferSize {
				jsonBuffer = ""
				t.safeSend(MessageData{
					Data: nil,
					Err: NewCLIJSONDecodeError(
						fmt.Sprintf("JSON message exceeded maximum buffer size of %d bytes", maxBufferSize),
						fmt.Errorf("buffer size exceeded"),
					),
				})
				continue
			}

			// Try to parse accumulated buffer
			var data map[string]any
			if err := json.Unmarshal([]byte(jsonBuffer), &data); err == nil {
				jsonBuffer = ""

				// Handle control responses separately
				if data["type"] == "control_response" {
					if response, ok := data["response"].(map[string]any); ok {
						if requestID, ok := response["request_id"].(string); ok {
							t.mu.Lock()
							t.pendingControlResponses[requestID] = response
							t.mu.Unlock()
						}
					}
					continue
				}

				t.safeSend(MessageData{Data: data, Err: nil})
			}
			// If JSON parsing failed, continue accumulating
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		t.safeSend(MessageData{Data: nil, Err: fmt.Errorf("error reading output: %w", err)})
	}
}

// readStderr reads and accumulates stderr output.
func (t *SubprocessCLITransport) readStderr() {
	defer t.taskGroup.Done()

	// Use a timer to enforce stderr timeout
	timer := time.NewTimer(stderrTimeout)
	defer timer.Stop()

	stderrLines := make([]string, 0)
	stderrSize := 0
	
	scanner := bufio.NewScanner(t.stderr)
	stderrChan := make(chan string, 100)
	done := make(chan bool)

	// Read stderr in a goroutine
	go func() {
		for scanner.Scan() {
			select {
			case stderrChan <- scanner.Text():
			case <-t.ctx.Done():
				return
			}
		}
		close(done)
	}()

	// Process stderr with size and timeout limits
	for {
		select {
		case line := <-stderrChan:
			lineSize := len(line)
			
			// Enforce memory limit
			if stderrSize+lineSize > maxStderrSize {
				stderrLines = append(stderrLines, fmt.Sprintf("[stderr truncated after %d bytes]", stderrSize))
				// Drain rest of stream without storing
				for {
					select {
					case <-stderrChan:
						// Continue draining
					case <-done:
						// Reading completed
						if len(stderrLines) > 0 {
							t.processStderr(stderrLines)
						}
						return
					}
				}
			}
			
			stderrLines = append(stderrLines, line)
			stderrSize += lineSize

		case <-done:
			// Reading completed
			if len(stderrLines) > 0 {
				t.processStderr(stderrLines)
			}
			return

		case <-timer.C:
			// Timeout reached
			stderrLines = append(stderrLines, fmt.Sprintf("[stderr collection timed out after %v]", stderrTimeout))
			t.processStderr(stderrLines)
			return

		case <-t.ctx.Done():
			return
		}
	}
}

// processStderr processes accumulated stderr output.
func (t *SubprocessCLITransport) processStderr(lines []string) {
	if len(lines) == 0 {
		return
	}

	stderrOutput := strings.Join(lines, "\n")
	
	// Wait for process exit code
	var exitCode int
	if t.cmd.ProcessState != nil {
		exitCode = t.cmd.ProcessState.ExitCode()
	}

	// Only treat as error if exit code is non-zero
	if exitCode != 0 {
		t.safeSend(MessageData{
			Data: nil,
			Err: NewProcessError(
				fmt.Sprintf("Command failed with exit code %d", exitCode),
				exitCode,
				stderrOutput,
			),
		})
	}
}

// sendControlRequest sends a control request and waits for response.
func (t *SubprocessCLITransport) sendControlRequest(ctx context.Context, request map[string]any) (map[string]any, error) {
	t.mu.Lock()
	if t.stdin == nil || !t.connected {
		t.mu.Unlock()
		return nil, NewCLIConnectionError("Not connected or stdin not available")
	}
	
	// Generate unique request ID
	requestID := fmt.Sprintf("req_%d_%d", atomic.AddUint64(&t.requestCounter, 1), time.Now().UnixNano())
	
	// Build control request
	controlRequest := map[string]any{
		"type":       "control_request",
		"request_id": requestID,
		"request":    request,
	}
	t.mu.Unlock()

	// Send request
	data, err := json.Marshal(controlRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal control request: %w", err)
	}

	select {
	case t.stdinChan <- append(data, '\n'):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Wait for response
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.mu.RLock()
			response, found := t.pendingControlResponses[requestID]
			t.mu.RUnlock()
			
			if found {
				t.mu.Lock()
				delete(t.pendingControlResponses, requestID)
				t.mu.Unlock()
				
				if subtype, ok := response["subtype"].(string); ok && subtype == "error" {
					if errMsg, ok := response["error"].(string); ok {
						return nil, NewCLIConnectionError(fmt.Sprintf("Control request failed: %s", errMsg))
					}
				}
				
				return response, nil
			}

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// stringPromptAdapter wraps a string prompt as a MessageStream
type stringPromptAdapter struct {
	prompt string
	sent   bool
}

func (s *stringPromptAdapter) Next(ctx context.Context) (map[string]any, error) {
	if s.sent {
		return nil, io.EOF
	}
	s.sent = true
	return map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": s.prompt,
		},
	}, nil
}

func (s *stringPromptAdapter) IsStreaming() bool {
	return false
}

// NewStringPromptStream creates a MessageStream from a string prompt
func NewStringPromptStream(prompt string) MessageStream {
	return &stringPromptAdapter{prompt: prompt}
}

// IsStringPrompt checks if a MessageStream is a simple string prompt
func IsStringPrompt(stream MessageStream) bool {
	_, ok := stream.(*stringPromptAdapter)
	return ok
}