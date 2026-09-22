//go:build !linux

package standalone

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/moby/moby/client"
)

var errUnsupportedPlatform = errors.New("standalone mode (DOCKER_STANDALONE) is only supported on Linux")

// NewAPIClient is not available on this platform.
func NewAPIClient(context.Context) (client.APIClient, error) {
	return nil, errUnsupportedPlatform
}

// MaybeReexecRootless is a no-op on this platform.
func MaybeReexecRootless() error {
	if Enabled() {
		return errUnsupportedPlatform
	}
	return nil
}

// RunLogger is not available on this platform.
func RunLogger([]string) {
	fmt.Fprintln(os.Stderr, errUnsupportedPlatform)
	os.Exit(1)
}
