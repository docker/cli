package task

import (
	"context"
	"testing"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func TestTaskPrintSorted(t *testing.T) {
	apiClient := &fakeClient{
		serviceInspectWithRaw: func(ref string, options types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			if ref == "service-id-one" {
				return *builders.Service(builders.ServiceName("service-name-1")), nil, nil
			}
			return *builders.Service(builders.ServiceName("service-name-10")), nil, nil
		},
	}

	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*builders.Task(
			builders.TaskID("id-foo"),
			builders.TaskServiceID("service-id-ten"),
			builders.TaskNodeID("id-node"),
			builders.WithTaskSpec(builders.TaskImage("myimage:mytag")),
			builders.TaskDesiredState(swarm.TaskStateReady),
			builders.WithStatus(builders.TaskState(swarm.TaskStateFailed), builders.Timestamp(time.Now().Add(-2*time.Hour))),
		),
		*builders.Task(
			builders.TaskID("id-bar"),
			builders.TaskServiceID("service-id-one"),
			builders.TaskNodeID("id-node"),
			builders.WithTaskSpec(builders.TaskImage("myimage:mytag")),
			builders.TaskDesiredState(swarm.TaskStateReady),
			builders.WithStatus(builders.TaskState(swarm.TaskStateFailed), builders.Timestamp(time.Now().Add(-2*time.Hour))),
		),
	}

	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, false), false, false, formatter.TableFormatKey)
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-sorted.golden")
}

func TestTaskPrintWithQuietOption(t *testing.T) {
	const quiet = true
	const trunc = false
	const noResolve = true
	apiClient := &fakeClient{}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{*builders.Task(builders.TaskID("id-foo"))}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, formatter.TableFormatKey)
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-quiet-option.golden")
}

func TestTaskPrintWithNoTruncOption(t *testing.T) {
	const quiet = false
	const trunc = false
	const noResolve = true
	apiClient := &fakeClient{}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*builders.Task(builders.TaskID("id-foo-yov6omdek8fg3k5stosyp2m50")),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .ID }}")
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-no-trunc-option.golden")
}

func TestTaskPrintWithGlobalService(t *testing.T) {
	const quiet = false
	const trunc = false
	const noResolve = true
	apiClient := &fakeClient{}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*builders.Task(builders.TaskServiceID("service-id-foo"), builders.TaskNodeID("node-id-bar"), builders.TaskSlot(0)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }}")
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-global-service.golden")
}

func TestTaskPrintWithReplicatedService(t *testing.T) {
	const quiet = false
	const trunc = false
	const noResolve = true
	apiClient := &fakeClient{}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*builders.Task(builders.TaskServiceID("service-id-foo"), builders.TaskSlot(1)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }}")
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-replicated-service.golden")
}

func TestTaskPrintWithIndentation(t *testing.T) {
	const quiet = false
	const trunc = false
	const noResolve = false
	apiClient := &fakeClient{
		serviceInspectWithRaw: func(ref string, options types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return *builders.Service(builders.ServiceName("service-name-foo")), nil, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *builders.Node(builders.NodeName("node-name-bar")), nil, nil
		},
	}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*builders.Task(
			builders.TaskID("id-foo"),
			builders.TaskServiceID("service-id-foo"),
			builders.TaskNodeID("id-node"),
			builders.WithTaskSpec(builders.TaskImage("myimage:mytag")),
			builders.TaskDesiredState(swarm.TaskStateReady),
			builders.WithStatus(builders.TaskState(swarm.TaskStateFailed), builders.Timestamp(time.Now().Add(-2*time.Hour))),
		),
		*builders.Task(
			builders.TaskID("id-bar"),
			builders.TaskServiceID("service-id-foo"),
			builders.TaskNodeID("id-node"),
			builders.WithTaskSpec(builders.TaskImage("myimage:mytag")),
			builders.TaskDesiredState(swarm.TaskStateReady),
			builders.WithStatus(builders.TaskState(swarm.TaskStateFailed), builders.Timestamp(time.Now().Add(-2*time.Hour))),
		),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, formatter.TableFormatKey)
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-indentation.golden")
}

func TestTaskPrintWithResolution(t *testing.T) {
	const quiet = false
	const trunc = false
	const noResolve = false
	apiClient := &fakeClient{
		serviceInspectWithRaw: func(ref string, options types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return *builders.Service(builders.ServiceName("service-name-foo")), nil, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *builders.Node(builders.NodeName("node-name-bar")), nil, nil
		},
	}
	cli := test.NewFakeCli(apiClient)
	tasks := []swarm.Task{
		*builders.Task(builders.TaskServiceID("service-id-foo"), builders.TaskSlot(1)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }} {{ .Node }}")
	assert.NilError(t, err)
	golden.Assert(t, cli.OutBuffer().String(), "task-print-with-resolution.golden")
}
