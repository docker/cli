package task

import (
	"context"
	"testing"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/internal/test"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/golden"
)

func TestTaskPrintWithQuietOption(t *testing.T) {
	quiet := true
	trunc := false
	noResolve := true
	apiClient := &fakeClient{}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{*Task(TaskID("id-foo"))}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, formatter.TableFormatKey)
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-quiet-option.golden")
}

func TestTaskPrintWithNoTruncOption(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := true
	apiClient := &fakeClient{}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*Task(TaskID("id-foo-yov6omdek8fg3k5stosyp2m50")),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .ID }}")
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-no-trunc-option.golden")
}

func TestTaskPrintWithGlobalService(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := true
	apiClient := &fakeClient{}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*Task(TaskServiceID("service-id-foo"), TaskNodeID("node-id-bar"), TaskSlot(0)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }}")
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-global-service.golden")
}

func TestTaskPrintWithReplicatedService(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := true
	apiClient := &fakeClient{}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*Task(TaskServiceID("service-id-foo"), TaskSlot(1)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }}")
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-replicated-service.golden")
}

func TestTaskPrintWithIndentation(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := false
	apiClient := &fakeClient{
		serviceInspectWithRaw: func(ref string, options types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return *Service(ServiceName("service-name-foo")), nil, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *Node(NodeName("node-name-bar")), nil, nil
		},
	}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*Task(
			TaskID("id-foo"),
			TaskServiceID("service-id-foo"),
			TaskNodeID("id-node"),
			WithTaskSpec(TaskImage("myimage:mytag")),
			TaskDesiredState(swarm.TaskStateReady),
			WithStatus(TaskState(swarm.TaskStateFailed), Timestamp(time.Now().Add(-2*time.Hour))),
		),
		*Task(
			TaskID("id-bar"),
			TaskServiceID("service-id-foo"),
			TaskNodeID("id-node"),
			WithTaskSpec(TaskImage("myimage:mytag")),
			TaskDesiredState(swarm.TaskStateReady),
			WithStatus(TaskState(swarm.TaskStateFailed), Timestamp(time.Now().Add(-2*time.Hour))),
		),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, formatter.TableFormatKey)
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-indentation.golden")
}

func TestTaskPrintWithResolution(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := false
	apiClient := &fakeClient{
		serviceInspectWithRaw: func(ref string, options types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return *Service(ServiceName("service-name-foo")), nil, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *Node(NodeName("node-name-bar")), nil, nil
		},
	}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*Task(TaskServiceID("service-id-foo"), TaskSlot(1)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }} {{ .Node }}")
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-resolution.golden")
}
