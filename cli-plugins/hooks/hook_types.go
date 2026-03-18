// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

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
	// which is currently being invoked. If a hook for `docker context` is
	// configured and the user executes `docker context ls`, the plugin will
	// be invoked with `context`.
	RootCmd      string            `json:"RootCmd,omitzero"`
	Flags        map[string]string `json:"Flags,omitzero"`
	CommandError string            `json:"CommandError,omitzero"`
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
