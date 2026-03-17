// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.25

// Package hooks defines the contract between the Docker CLI and CLI plugin hook
// implementations.
//
// # Audience
//
// This package is intended to be imported by CLI plugin implementations that
// implement a "hooks" subcommand, and by the Docker CLI when invoking those
// hooks.
//
// # Contract and wire format
//
// Hook inputs (see [Request]) are serialized as JSON and passed to the plugin hook
// subcommand (currently as a command-line argument). Hook outputs are emitted by
// the plugin as JSON (see [Response]).
//
// # Stability
//
// The types that represent the hook contract ([Request], [Response] and related
// constants) are considered part of Docker CLI's public Go API.
// Fields and values may be extended in a backwards-compatible way (for example,
// adding new fields), but existing fields and their meaning should remain stable.
// Plugins should ignore unknown fields and unknown hook types to remain
// forwards-compatible.
package hooks

// ResponseType is the type of response from the plugin.
type ResponseType int

const (
	NextSteps ResponseType = 0
)

// Request is the type representing the information
// that plugins declaring support for hooks get passed when
// being invoked following a CLI command execution.
type Request struct {
	// RootCmd is a string representing the matching hook configuration
	// which is currently being invoked. If a hook for "docker context"
	// is configured and the user executes "docker context ls", the plugin
	// is invoked with "context".
	RootCmd string `json:"RootCmd,omitzero"`

	// Flags contains flags that were set on the command for which the
	// hook was invoked. It uses flag names as key, with leading hyphens
	// removed ("--flag" and "-flag" are included as "flag" and "f").
	//
	// Flag values are not included and are set to an empty string,
	// except for boolean flags known to the CLI itself, for which
	// the value is either "true", or "false".
	//
	// Plugins can use this information to adjust their [Response]
	// based on whether the command triggering the hook was invoked
	// with.
	Flags map[string]string `json:"Flags,omitzero"`

	// CommandError is a string containing the error output (if any)
	// of the command for which the hook was invoked.
	CommandError string `json:"CommandError,omitzero"`
}

// Response represents a plugin hook response. Plugins
// declaring support for CLI hooks need to print a JSON
// representation of this type when their hook subcommand
// is invoked.
type Response struct {
	Type     ResponseType `json:"Type"`
	Template string       `json:"Template,omitzero"`
}

// HookType is the type of response from the plugin.
//
// Deprecated: use [ResponseType] instead.
//
//go:fix inline
type HookType = ResponseType

// HookMessage represents a plugin hook response.
//
// Deprecated: use [Response] instead.
//
//go:fix inline
type HookMessage = Response
