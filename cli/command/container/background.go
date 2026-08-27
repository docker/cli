package container

import (
	"io"

	"github.com/docker/cli/cli/streams"
)

// stdinForAttach returns stdin for hijacking, or nil when copying it would
// spin (a background job whose stdin is still the controlling TTY).
func stdinForAttach(in *streams.In) io.ReadCloser {
	if stdinFromBackgroundJob(in) {
		return nil
	}
	return in
}
