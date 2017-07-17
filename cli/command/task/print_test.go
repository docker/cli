package task

import (
	"bytes"
	"testing"
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/cli/internal/test"
	"golang.org/x/net/context"
	// Import builders to get the builder function as package function
	. "github.com/docker/cli/cli/internal/test/builders"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/testutil"
	"github.com/docker/docker/pkg/testutil/golden"
	"github.com/stretchr/testify/assert"
)

func TestTaskPrintWithQuietOption(t *testing.T) {
	quiet := true
	trunc := false
	noResolve := true
	buf := new(bytes.Buffer)
	apiClient := &fakeClient{}
	cli := test.NewFakeCliWithOutput(apiClient, buf)
	tasks := []swarm.Task{
		*Task(TaskID("id-foo")),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, formatter.TableFormatKey)
	assert.NoError(t, err)
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "task-print-with-quiet-option.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestTaskPrintWithNoTruncOption(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := true
	buf := new(bytes.Buffer)
	apiClient := &fakeClient{}
	cli := test.NewFakeCliWithOutput(apiClient, buf)
	tasks := []swarm.Task{
		*Task(TaskID("id-foo-yov6omdek8fg3k5stosyp2m50")),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .ID }}")
	assert.NoError(t, err)
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "task-print-with-no-trunc-option.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestTaskPrintWithGlobalService(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := true
	buf := new(bytes.Buffer)
	apiClient := &fakeClient{}
	cli := test.NewFakeCliWithOutput(apiClient, buf)
	tasks := []swarm.Task{
		*Task(TaskServiceID("service-id-foo"), TaskNodeID("node-id-bar"), TaskSlot(0)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }}")
	assert.NoError(t, err)
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "task-print-with-global-service.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestTaskPrintWithReplicatedService(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := true
	buf := new(bytes.Buffer)
	apiClient := &fakeClient{}
	cli := test.NewFakeCliWithOutput(apiClient, buf)
	tasks := []swarm.Task{
		*Task(TaskServiceID("service-id-foo"), TaskSlot(1)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }}")
	assert.NoError(t, err)
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "task-print-with-replicated-service.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestTaskPrintWithIndentation(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := false
	buf := new(bytes.Buffer)
	apiClient := &fakeClient{
		serviceInspectWithRaw: func(ref string, options types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return *Service(ServiceName("service-name-foo")), nil, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *Node(NodeName("node-name-bar")), nil, nil
		},
	}
	cli := test.NewFakeCliWithOutput(apiClient, buf)
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
	assert.NoError(t, err)
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "task-print-with-indentation.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}

func TestTaskPrintWithResolution(t *testing.T) {
	quiet := false
	trunc := false
	noResolve := false
	buf := new(bytes.Buffer)
	apiClient := &fakeClient{
		serviceInspectWithRaw: func(ref string, options types.ServiceInspectOptions) (swarm.Service, []byte, error) {
			return *Service(ServiceName("service-name-foo")), nil, nil
		},
		nodeInspectWithRaw: func(ref string) (swarm.Node, []byte, error) {
			return *Node(NodeName("node-name-bar")), nil, nil
		},
	}
	cli := test.NewFakeCliWithOutput(apiClient, buf)
	tasks := []swarm.Task{
		*Task(TaskServiceID("service-id-foo"), TaskSlot(1)),
	}
	err := Print(context.Background(), cli, tasks, idresolver.New(apiClient, noResolve), trunc, quiet, "{{ .Name }} {{ .Node }}")
	assert.NoError(t, err)
	actual := buf.String()
	expected := golden.Get(t, []byte(actual), "task-print-with-resolution.golden")
	testutil.EqualNormalizedString(t, testutil.RemoveSpace, actual, string(expected))
}
