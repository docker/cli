package node // import "docker.com/cli/v28/cli/command/node"

import (
	"github.com/docker/cli/v28/opts"
)

type nodeOptions struct {
	annotations
	role         string
	availability string
}

type annotations struct {
	labels opts.ListOpts
}

func newNodeOptions() *nodeOptions {
	return &nodeOptions{
		annotations: annotations{
			labels: opts.NewListOpts(nil),
		},
	}
}
