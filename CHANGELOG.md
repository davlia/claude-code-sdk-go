# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of Claude Code SDK for Go
- Core `Query` function for simple, stateless interactions
- `Client` type for bidirectional, stateful conversations
- Support for all Claude Code message types
- Support for tool usage (Read, Write, Bash, etc.)
- MCP server configuration support
- Comprehensive error handling
- Example applications demonstrating usage
- Full test coverage
- Go modules support
- MIT License

### Features
- Async message streaming using channels
- Context-based cancellation support
- Custom message stream interface
- Permission mode configuration
- Working directory configuration
- System prompt customization
- Max turns limitation
- Tool allowlist/denylist
- Cost tracking in result messages

### Documentation
- Comprehensive README with examples
- Inline godoc documentation
- Contributing guidelines
- Example applications

[Unreleased]: https://github.com/davlia/claude-code-sdk-go/commits/main