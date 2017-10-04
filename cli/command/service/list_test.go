package service

import (
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestServiceListOrder(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		serviceListFunc: func(ctx context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
			return []swarm.Service{
				newService("a57dbe8", "service-1-foo"),
				newService("a57dbdd", "service-10-foo"),
				newService("aaaaaaa", "service-2-foo"),
			}, nil
		},
	})
	cmd := newListCommand(cli)
	cmd.Flags().Set("format", "{{.Name}}")
	assert.NoError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "service-list-sort.golden")
}
