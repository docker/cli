// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.21

package manager

import (
	"fmt"
)

// pluginError is set as Plugin.Err by NewPlugin if the plugin
// candidate fails one of the candidate tests. This exists primarily
// to implement encoding.TextMarshaller such that rendering a plugin as JSON
// (e.g. for `docker info -f '{{json .CLIPlugins}}'`) renders the Err
// field as a useful string and not just `{}`. See
// https://github.com/golang/go/issues/10748 for some discussion
// around why the builtin error type doesn't implement this.
type pluginError struct {
	cause error
}

// Error satisfies the core error interface for pluginError.
func (e *pluginError) Error() string {
	return e.cause.Error()
}

// Cause satisfies the github.com/pkg/errors.causer interface for pluginError.
// TODO: remove this once all users switch away from github.com/pkg/errors.
func (e *pluginError) Cause() error {
	return e.cause
}

// Unwrap provides compatibility for Go 1.13+ error chains.
func (e *pluginError) Unwrap() error {
	return e.cause
}

// MarshalText marshalls the pluginError into a textual form.
func (e *pluginError) MarshalText() (text []byte, err error) {
	return []byte(e.cause.Error()), nil
}

// NewPluginError creates a new pluginError, analogous to
// [fmt.Errorf].
func NewPluginError(msg string, args ...any) error {
	return &pluginError{cause: fmt.Errorf(msg, args...)}
}
