package debug

import (
	"os"

	"github.com/containerd/log"
	"go.opentelemetry.io/otel"
)

// Enable sets the DEBUG env var to true
// and makes the logger to log at debug level.
func Enable() {
	_ = os.Setenv("DEBUG", "1")
	_ = log.SetLevel("debug")
}

// Disable sets the DEBUG env var to false
// and makes the logger to log at info level.
func Disable() {
	_ = os.Setenv("DEBUG", "")
	_ = log.SetLevel("info")
}

// IsEnabled checks whether the debug flag is set or not.
func IsEnabled() bool {
	return os.Getenv("DEBUG") != ""
}

// OTELErrorHandler is an error handler for OTEL that
// uses the CLI debug package to log messages when an error
// occurs.
//
// The default is to log to the debug level which is only
// enabled when debugging is enabled.
var OTELErrorHandler otel.ErrorHandler = otel.ErrorHandlerFunc(func(err error) {
	if err == nil {
		return
	}
	log.L.WithError(err).Debug("otel error")
})
