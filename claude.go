// Package claude provides a Go SDK for Claude Code.
//
// See README.md and the examples directory for usage information.
package claude

// This file exists to provide a clear entry point for the package
// and ensure all exported types are available.

// Re-export key functions at package level for convenience
var (
	// Version is re-exported from version.go
	_ = Version
)
