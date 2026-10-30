package container

import "github.com/docker/cli/cli/streams"

func stdinFromBackgroundJob(_ *streams.In) bool {
	return false
}
