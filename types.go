package claude

import "fmt"

// PermissionMode defines how tool permissions are handled
type PermissionMode string

const (
	// PermissionModeDefault prompts for dangerous tools
	PermissionModeDefault PermissionMode = "default"
	// PermissionModeAcceptEdits auto-accepts file edits
	PermissionModeAcceptEdits PermissionMode = "acceptEdits"
	// PermissionModeBypassPermissions allows all tools (use with caution)
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
)

// MCPServerType defines the type of MCP server
type MCPServerType string

const (
	MCPServerTypeStdio MCPServerType = "stdio"
	MCPServerTypeSSE   MCPServerType = "sse"
	MCPServerTypeHTTP  MCPServerType = "http"
)

// MCPServerConfig is the interface for all MCP server configurations
type MCPServerConfig interface {
	GetType() MCPServerType
}

// MCPStdioServerConfig represents MCP stdio server configuration
type MCPStdioServerConfig struct {
	Type    MCPServerType     `json:"type,omitempty"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

func (c MCPStdioServerConfig) GetType() MCPServerType {
	if c.Type == "" {
		return MCPServerTypeStdio
	}
	return c.Type
}

// MCPSSEServerConfig represents MCP SSE server configuration
type MCPSSEServerConfig struct {
	Type    MCPServerType     `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (c MCPSSEServerConfig) GetType() MCPServerType {
	return c.Type
}

// MCPHTTPServerConfig represents MCP HTTP server configuration
type MCPHTTPServerConfig struct {
	Type    MCPServerType     `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (c MCPHTTPServerConfig) GetType() MCPServerType {
	return c.Type
}

// ContentBlock is the interface for all content block types
type ContentBlock interface {
	contentBlock()
}

// TextBlock represents text content
type TextBlock struct {
	Text string `json:"text"`
}

func (TextBlock) contentBlock() {}

// ToolUseBlock represents tool usage
type ToolUseBlock struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

func (ToolUseBlock) contentBlock() {}

// ToolResultBlock represents tool result
type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content,omitempty"` // string or []map[string]any
	IsError   *bool  `json:"is_error,omitempty"`
}

func (ToolResultBlock) contentBlock() {}

// Message is the interface for all message types
type Message interface {
	message()
}

// UserMessage represents a user message
type UserMessage struct {
	Content string `json:"content"`
}

func (UserMessage) message() {}

// AssistantMessage represents an assistant message with content blocks
type AssistantMessage struct {
	Content []ContentBlock `json:"content"`
}

func (AssistantMessage) message() {}

// SystemMessage represents a system message with metadata
type SystemMessage struct {
	Subtype string         `json:"subtype"`
	Data    map[string]any `json:"data"`
}

func (SystemMessage) message() {}

// ResultMessage represents a result message with cost and usage information
type ResultMessage struct {
	Subtype       string         `json:"subtype"`
	DurationMS    int            `json:"duration_ms"`
	DurationAPIMS int            `json:"duration_api_ms"`
	IsError       bool           `json:"is_error"`
	NumTurns      int            `json:"num_turns"`
	SessionID     string         `json:"session_id"`
	TotalCostUSD  *float64       `json:"total_cost_usd,omitempty"`
	Usage         map[string]any `json:"usage,omitempty"`
	Result        *string        `json:"result,omitempty"`
}

func (ResultMessage) message() {}

// parseMessage parses a message from raw JSON data.
func parseMessage(data map[string]any) (Message, error) {
	msgType, ok := data["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid message type")
	}

	switch msgType {
	case "user":
		return parseUserMessage(data)
	case "assistant":
		return parseAssistantMessage(data)
	case "system":
		return parseSystemMessage(data), nil
	case "result":
		return parseResultMessage(data), nil
	default:
		return nil, fmt.Errorf("unknown message type: %s", msgType)
	}
}

func parseUserMessage(data map[string]any) (*UserMessage, error) {
	content, ok := data["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user message content")
	}
	return &UserMessage{Content: content}, nil
}

func parseAssistantMessage(data map[string]any) (*AssistantMessage, error) {
	contentData, ok := data["content"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid assistant message content")
	}

	var content []ContentBlock
	for _, item := range contentData {
		block, err := parseContentBlock(item)
		if err != nil {
			return nil, err
		}
		content = append(content, block)
	}

	return &AssistantMessage{Content: content}, nil
}

func parseContentBlock(data any) (ContentBlock, error) {
	blockData, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid content block format")
	}

	blockType, ok := blockData["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing content block type")
	}

	switch blockType {
	case "text":
		text, ok := blockData["text"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid text block content")
		}
		return &TextBlock{Text: text}, nil

	case "tool_use":
		id, _ := blockData["id"].(string)
		name, _ := blockData["name"].(string)
		input, _ := blockData["input"].(map[string]any)
		return &ToolUseBlock{ID: id, Name: name, Input: input}, nil

	case "tool_result":
		toolUseID, _ := blockData["tool_use_id"].(string)
		content := blockData["content"]

		var isError *bool
		if val, ok := blockData["is_error"].(bool); ok {
			isError = &val
		}

		return &ToolResultBlock{
			ToolUseID: toolUseID,
			Content:   content,
			IsError:   isError,
		}, nil

	default:
		return nil, fmt.Errorf("unknown content block type: %s", blockType)
	}
}

func parseSystemMessage(data map[string]any) *SystemMessage {
	subtype, _ := data["subtype"].(string)
	msgData, _ := data["data"].(map[string]any)
	return &SystemMessage{Subtype: subtype, Data: msgData}
}

func parseResultMessage(data map[string]any) *ResultMessage {
	msg := &ResultMessage{}

	msg.Subtype, _ = data["subtype"].(string)

	if val, ok := data["duration_ms"].(float64); ok {
		msg.DurationMS = int(val)
	}

	if val, ok := data["duration_api_ms"].(float64); ok {
		msg.DurationAPIMS = int(val)
	}

	msg.IsError, _ = data["is_error"].(bool)

	if val, ok := data["num_turns"].(float64); ok {
		msg.NumTurns = int(val)
	}

	msg.SessionID, _ = data["session_id"].(string)

	if val, ok := data["total_cost_usd"].(float64); ok {
		msg.TotalCostUSD = &val
	}

	msg.Usage, _ = data["usage"].(map[string]any)

	if val, ok := data["result"].(string); ok {
		msg.Result = &val
	}

	return msg
}

// MessageResult wraps a message or error for channel communication
type MessageResult struct {
	Message Message
	Error   error
}
