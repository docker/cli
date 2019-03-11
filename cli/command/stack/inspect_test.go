package stack

import (
	"io/ioutil"
	"testing"

	"github.com/docker/cli/internal/test"
	composetypes "github.com/docker/stacks/pkg/compose/types"
	stacktypes "github.com/docker/stacks/pkg/types"
	"gotest.tools/assert"
	"gotest.tools/golden"
)

func TestStackInspectServerSideSuccess(t *testing.T) {
	stacks := []stacktypes.Stack{
		{
			Orchestrator: "swarm",
			Spec: stacktypes.StackSpec{
				Metadata: stacktypes.Metadata{
					Name: "stackname",
				},
				Collection: "collection",
				Services: composetypes.Services{
					composetypes.ServiceConfig{
						Name: "svc1",
					},
				},
			},
			StackResources: stacktypes.StackResources{
				Services: map[string]stacktypes.StackResource{
					"svc1": {
						Orchestrator: "swarm",
						Kind:         "service",
						ID:           "id",
					},
				},
			},
			Status: stacktypes.StackStatus{
				ServicesStatus: map[string]stacktypes.ServiceStatus{
					"svc1": {
						DesiredTasks: 1,
						RunningTasks: 1,
					},
				},
			},
		},
	}
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		stackListFunc: func(options stacktypes.StackListOptions) ([]stacktypes.Stack, error) {
			return stacks, nil
		},
	})
	cmd := newInspectCommand(cli, &orchestrator)
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"stackname"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-inspect-server-side.golden")
}
