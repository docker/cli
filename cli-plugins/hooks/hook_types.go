package hooks

// Request is the type representing the information
// that plugins declaring support for hooks get passed when
// being invoked following a CLI command execution.
type Request struct {
	// RootCmd is a string representing the matching hook configuration
	// which is currently being invoked. If a hook for `docker context` is
	// configured and the user executes `docker context ls`, the plugin will
	// be invoked with `context`.
	RootCmd      string
	Flags        map[string]string
	CommandError string
}
