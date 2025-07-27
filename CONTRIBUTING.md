# Contributing to Claude Code SDK for Go

We welcome contributions to the Claude Code SDK for Go! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/claude-code-sdk-go.git`
3. Create a new branch: `git checkout -b feature/your-feature-name`
4. Make your changes
5. Run tests: `go test ./...`
6. Commit your changes: `git commit -am 'Add new feature'`
7. Push to your fork: `git push origin feature/your-feature-name`
8. Create a pull request

## Development Setup

### Prerequisites

- Go 1.21 or later
- Node.js (for Claude Code CLI)
- Claude Code CLI: `npm install -g @anthropic-ai/claude-code`

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

### Code Style

This project follows standard Go conventions:

- Use `gofmt` to format your code
- Use `golint` for linting
- Use `go vet` for static analysis
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines

### Pre-commit Checklist

Before submitting a pull request, ensure:

- [ ] All tests pass
- [ ] Code is formatted with `gofmt`
- [ ] No `golint` warnings
- [ ] No `go vet` warnings
- [ ] New features have tests
- [ ] Documentation is updated
- [ ] Examples work correctly

## Code Organization

- `*.go` - Main package files
- `examples/` - Example applications
- `internal/` - Internal packages (if needed)
- `testdata/` - Test fixtures (if needed)

## Testing Guidelines

- Write table-driven tests where appropriate
- Test both success and error cases
- Use subtests for better organization
- Mock external dependencies
- Aim for >80% test coverage

## Documentation

- Add godoc comments to all exported types and functions
- Include examples in documentation where helpful
- Update README.md for significant changes
- Keep examples up to date

## Commit Messages

Follow conventional commit format:

```
type(scope): description

body (optional)

footer (optional)
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test changes
- `refactor`: Code refactoring
- `style`: Code style changes
- `chore`: Maintenance tasks

## Pull Request Process

1. Ensure your PR description clearly describes the problem and solution
2. Include the relevant issue number if applicable
3. Update documentation as needed
4. Add tests for new functionality
5. Ensure all checks pass
6. Request review from maintainers

## Reporting Issues

When reporting issues, please include:

- Go version (`go version`)
- Claude Code version (`claude-code --version`)
- Operating system and version
- Minimal reproducible example
- Expected vs actual behavior
- Any error messages or logs

## Community

- Be respectful and constructive
- Help others when you can
- Follow the [Go Code of Conduct](https://golang.org/conduct)

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

Thank you for contributing!