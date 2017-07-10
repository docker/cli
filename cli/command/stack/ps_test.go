package stack

import (
	"bytes"
	"io/ioutil"
	"testing"
	"time"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/internal/test"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/testutil"
	"github.com/docker/docker/pkg/testutil/golden"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
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
		}, &bytes.Buffer{}))
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		testutil.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestStackPsEmptyStack(t *testing.T) {
	out := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	fakeCli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{}, nil
		},
	}, out)
	fakeCli.SetErr(stderr)
	cmd := newPsCommand(fakeCli)
	cmd.SetArgs([]string{"foo"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "", out.String())
	assert.Equal(t, "Nothing found in stack: foo\n", stderr.String())
}

func TestStackPsWithQuietOption(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(TaskID("id-foo"))}, nil
		},
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("quiet", "true")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-ps-with-quiet-option.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))

}

func TestStackPsWithNoTruncOption(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(TaskID("xn4cypcov06f2w8gsbaf2lst3"))}, nil
		},
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("no-trunc", "true")
	cmd.Flags().Set("format", "{{ .ID }}")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-ps-with-no-trunc-option.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestStackPsWithNoResolveOption(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(
				TaskNodeID("id-node-foo"),
			)}, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *Node(NodeName("node-name-bar")), nil, nil
		},
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("no-resolve", "true")
	cmd.Flags().Set("format", "{{ .Node }}")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-ps-with-no-resolve-option.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestStackPsWithFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(TaskServiceID("service-id-foo"))}, nil
		},
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	cmd.Flags().Set("format", "{{ .Name }}")
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-ps-with-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestStackPsWithConfigFormat(t *testing.T) {
	buf := new(bytes.Buffer)
	cli := test.NewFakeCli(&fakeClient{
		taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
			return []swarm.Task{*Task(TaskServiceID("service-id-foo"))}, nil
		},
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{
		TasksFormat: "{{ .Name }}",
	})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-ps-with-config-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestStackPsWithoutFormat(t *testing.T) {
	buf := new(bytes.Buffer)
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
	}, buf)
	cli.SetConfigfile(&configfile.ConfigFile{})
	cmd := newPsCommand(cli)
	cmd.SetArgs([]string{"foo"})
	assert.NoError(t, cmd.Execute())
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "stack-ps-without-format.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}
