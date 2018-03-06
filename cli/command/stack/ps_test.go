package stack

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/test"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/golden"
	"github.com/pkg/errors"
)

func TestStackPsErrors(t *testing.T) {
	testCases := []struct {
		args          []string
		taskListFunc  func(options types.TaskListOptions) ([]swarm.Task, error)
		expectedError string
	}{

		{
			args:          []string{},
			expectedError: "requires exactly 1 argument",
		},
		{
			args:          []string{"foo", "bar"},
			expectedError: "requires exactly 1 argument",
		},
		{
			args: []string{"foo"},
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return nil, errors.Errorf("error getting tasks")
			},
			expectedError: "error getting tasks",
		},
	}

	for _, tc := range testCases {
		cmd := newPsCommand(test.NewFakeCli(&fakeClient{
			taskListFunc: tc.taskListFunc,
		}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestStackPsEmptyStack(t *testing.T) {
	fakeCli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{}, nil
		},
	})
	cmd := newPsCommand(fakeCli)
	cmd.SetArgs([]string{"foo"})
	cmd.SetOutput(ioutil.Discard)

	assert.Error(t, cmd.Execute(), "nothing found in stack: foo")
	assert.Check(t, is.Equal("", fakeCli.OutBuffer().String()))
}

func TestStackPsWithQuietOption(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(TaskID("id-foo"))}, nil
		},
	})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("quiet", "true")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-ps-with-quiet-option.golden")

}

func TestStackPsWithNoTruncOption(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(TaskID("xn4cypcov06f2w8gsbaf2lst3"))}, nil
		},
	})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("no-trunc", "true")
	cmd.Flags().Set("format", "{{ .ID }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-ps-with-no-trunc-option.golden")
}

func TestStackPsWithNoResolveOption(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(
				TaskNodeID("id-node-foo"),
			)}, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *Node(NodeName("node-name-bar")), nil, nil
		},
	})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("no-resolve", "true")
	cmd.Flags().Set("format", "{{ .Node }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-ps-with-no-resolve-option.golden")
}

func TestStackPsWithFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(TaskServiceID("service-id-foo"))}, nil
		},
	})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("format", "{{ .Name }}")
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-ps-with-format.golden")
}

func TestStackPsWithConfigFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(TaskServiceID("service-id-foo"))}, nil
		},
	})
	cli.SetConfigFile(&configfile.ConfigFile{
		TasksFormat: "{{ .Name }}",
	})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-ps-with-config-format.golden")
}

func TestStackPsWithoutFormat(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(
				TaskID("id-foo"),
				TaskServiceID("service-id-foo"),
				TaskNodeID("id-node"),
				WithTaskSpec(TaskImage("myimage:mytag")),
				TaskDesiredState(swarm.TaskStateReady),
				WithStatus(TaskState(swarm.TaskStateFailed), Timestamp(time.Now().Add(-2*time.Hour))),
			)}, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *Node(NodeName("node-name-bar")), nil, nil
		},
	})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.NilError(t, cmd.Execute())
	golden.Assert(t, cli.OutBuffer().String(), "stack-ps-without-format.golden")
}
