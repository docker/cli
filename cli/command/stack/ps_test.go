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
	"github.com/pkg/errors"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/golden"
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
		}), &orchestrator)
		cmd.SetArgs(tc.args)
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), tc.expectedError)
	}
}

func TestStackPs(t *testing.T) {
	testCases := []struct {
		doc                string
		taskListFunc       func(types.TaskListOptions) ([]swarm.Task, error)
		nodeInspectWithRaw func(string) (swarm.Node, []byte, error)
		config             configfile.ConfigFile
		args               []string
		flags              map[string]string
		expectedErr        string
		golden             string
	}{
		{
			doc:         "WithEmptyName",
			args:        []string{"'   '"},
			expectedErr: `invalid stack name: "'   '"`,
		},
		{
			doc: "WithEmptyStack",
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{}, nil
			},
			args:        []string{"foo"},
			expectedErr: "nothing found in stack: foo",
		},
		{
			doc: "WithQuietOption",
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{*Task(TaskID("id-foo"))}, nil
			},
			args: []string{"foo"},
			flags: map[string]string{
				"quiet": "true",
			},
			golden: "stack-ps-with-quiet-option.golden",
		},
		{
			doc: "WithNoTruncOption",
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{*Task(TaskID("xn4cypcov06f2w8gsbaf2lst3"))}, nil
			},
			args: []string{"foo"},
			flags: map[string]string{
				"no-trunc": "true",
				"format":   "{{ .ID }}",
			},
			golden: "stack-ps-with-no-trunc-option.golden",
		},
		{
			doc: "WithNoResolveOption",
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{*Task(
					TaskNodeID("id-node-foo"),
				)}, nil
			},
			nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
				return *Node(NodeName("node-name-bar")), nil, nil
			},
			args: []string{"foo"},
			flags: map[string]string{
				"no-resolve": "true",
				"format":     "{{ .Node }}",
			},
			golden: "stack-ps-with-no-resolve-option.golden",
		},
		{
			doc: "WithFormat",
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{*Task(TaskServiceID("service-id-foo"))}, nil
			},
			args: []string{"foo"},
			flags: map[string]string{
				"format": "{{ .Name }}",
			},
			golden: "stack-ps-with-format.golden",
		},
		{
			doc: "WithConfigFormat",
			taskListFunc: func(options types.TaskListOptions) ([]swarm.Task, error) {
				return []swarm.Task{*Task(TaskServiceID("service-id-foo"))}, nil
			},
			config: configfile.ConfigFile{
				TasksFormat: "{{ .Name }}",
			},
			args:   []string{"foo"},
			golden: "stack-ps-with-config-format.golden",
		},
		{
			doc: "WithoutFormat",
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
			args:   []string{"foo"},
			golden: "stack-ps-without-format.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			cli := test.NewFakeCli(&fakeClient{
				taskListFunc:       tc.taskListFunc,
				nodeInspectWithRaw: tc.nodeInspectWithRaw,
			})
			cli.SetConfigFile(&tc.config)

			cmd := newPsCommand(cli, &orchestrator)
			cmd.SetArgs(tc.args)
			for key, value := range tc.flags {
				cmd.Flags().Set(key, value)
			}
			cmd.SetOutput(ioutil.Discard)

			if tc.expectedErr != "" {
				assert.Error(t, cmd.Execute(), tc.expectedErr)
				assert.Check(t, is.Equal("", cli.OutBuffer().String()))
				return
			}
			assert.NilError(t, cmd.Execute())
			golden.Assert(t, cli.OutBuffer().String(), tc.golden)
		})
	}
}
