package service

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/docker/cli/cli/internal/test"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/testutil"
	"github.com/docker/docker/pkg/testutil/golden"
	"github.com/stretchr/testify/assert"
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
	actual := cli.OutBuffer().String()
	expected := golden.Get(t, []byte(actual), "service-list-sort.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}
