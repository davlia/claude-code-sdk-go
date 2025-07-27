package claude

import "encoding/json"

// Options represents the configuration for a Claude Code client.
type Options struct {
	AllowedTools             []string                   `json:"allowed_tools,omitempty"`
	MaxThinkingTokens        int                        `json:"max_thinking_tokens,omitempty"`
	SystemPrompt             string                     `json:"system_prompt,omitempty"`
	AppendSystemPrompt       string                     `json:"append_system_prompt,omitempty"`
	MCPTools                 []string                   `json:"mcp_tools,omitempty"`
	MCPServers               map[string]MCPServerConfig `json:"mcp_servers,omitempty"`
	PermissionMode           PermissionMode             `json:"permission_mode,omitempty"`
	ContinueConversation     bool                       `json:"continue_conversation,omitempty"`
	Resume                   string                     `json:"resume,omitempty"`
	MaxTurns                 *int                       `json:"max_turns,omitempty"`
	DisallowedTools          []string                   `json:"disallowed_tools,omitempty"`
	Model                    string                     `json:"model,omitempty"`
	PermissionPromptToolName string                     `json:"permission_prompt_tool_name,omitempty"`
	Cwd                      string                     `json:"cwd,omitempty"`
}

// NewOptions creates Options with default values.
func NewOptions() *Options {
	return &Options{
		MaxThinkingTokens: 8000,
		AllowedTools:      []string{},
		MCPTools:          []string{},
		MCPServers:        make(map[string]MCPServerConfig),
		DisallowedTools:   []string{},
	}
}

// MarshalJSON customizes JSON marshaling for Options.
func (o Options) MarshalJSON() ([]byte, error) {
	type optionsAlias Options

	// Convert MCPServerConfig to the format expected by the CLI
	mcpServers := make(map[string]map[string]any)
	for name, config := range o.MCPServers {
		switch c := config.(type) {
		case MCPStdioServerConfig:
			mcpServers[name] = map[string]any{
				"type":    c.GetType(),
				"command": c.Command,
				"args":    c.Args,
				"env":     c.Env,
			}
		case MCPSSEServerConfig:
			mcpServers[name] = map[string]any{
				"type": c.GetType(),
				"url":  c.URL,
			}
		}
	}

	return json.Marshal(&struct {
		optionsAlias
		MCPServers map[string]map[string]any `json:"mcp_servers,omitempty"`
	}{
		optionsAlias: optionsAlias(o),
		MCPServers:   mcpServers,
	})
}
