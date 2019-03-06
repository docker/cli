package stack

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	stacktypes "github.com/docker/stacks/pkg/types"
	"gotest.tools/assert"
)

func TestDeployWithEmptyName(t *testing.T) {
	cmd := newDeployCommand(test.NewFakeCli(&fakeClient{}), nil)
	cmd.SetArgs([]string{"'   '"})
	cmd.SetOutput(ioutil.Discard)

	assert.ErrorContains(t, cmd.Execute(), `invalid stack name: "'   '"`)
}

func TestDeployWithParseFailure(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		parseComposeInputFunc: func(input stacktypes.ComposeInput) (*stacktypes.StackCreate, error) {
			return &stacktypes.StackCreate{}, fmt.Errorf("malformed compose file")
		},
	})
	composefile := bytes.NewBufferString(`
This is not a legal compose file
`)
	cli.SetIn(streams.NewIn(ioutil.NopCloser(composefile)))
	cmd := newDeployCommand(cli, nil)
	cmd.SetArgs([]string{"testname"})
	cmd.Flags().Set("compose-file", "-")
	cmd.SetOutput(ioutil.Discard)

	assert.ErrorContains(t, cmd.Execute(), `malformed compose file`)
}

func TestDeployWithCreateFailure(t *testing.T) {
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		parseComposeInputFunc: func(input stacktypes.ComposeInput) (*stacktypes.StackCreate, error) {
			return &stacktypes.StackCreate{}, nil
		},
		stackCreateFunc: func(stack stacktypes.StackCreate, options stacktypes.StackCreateOptions) (stacktypes.StackCreateResponse, error) {
			return stacktypes.StackCreateResponse{}, fmt.Errorf("failed to create stack")
		},
	})
	composefile := bytes.NewBufferString(`
version: "3.0"
services:
  web:
    image: busybox
`)
	cli.SetIn(streams.NewIn(ioutil.NopCloser(composefile)))
	cmd := newDeployCommand(cli, nil)
	cmd.SetArgs([]string{"testname"})
	cmd.Flags().Set("compose-file", "-")
	cmd.SetOutput(ioutil.Discard)

	assert.ErrorContains(t, cmd.Execute(), `failed to create stack`)
}

func TestDeployUpdateExistingOutOfSequence(t *testing.T) {
	stacks := []stacktypes.Stack{
		{
			Metadata: stacktypes.Metadata{
				Name: "stackname",
			},
			Orchestrator: "swarm",
			Spec: stacktypes.StackSpec{
				Collection: "collection",
			},
		},
	}
	cli := test.NewFakeCli(&fakeClient{
		version: clientSideStackVersion,
		stackListFunc: func(options stacktypes.StackListOptions) ([]stacktypes.Stack, error) {
			return stacks, nil
		},
		parseComposeInputFunc: func(input stacktypes.ComposeInput) (*stacktypes.StackCreate, error) {
			return &stacktypes.StackCreate{}, nil
		},
		stackUpdateFunc: func(id string, version stacktypes.Version, spec stacktypes.StackSpec, options stacktypes.StackUpdateOptions) error {

			return fmt.Errorf("update out of sequence")
		},
	})
	composefile := bytes.NewBufferString(`
version: "3.0"
services:
  web:
    image: busybox
`)
	cli.SetIn(streams.NewIn(ioutil.NopCloser(composefile)))
	cmd := newDeployCommand(cli, nil)
	cmd.SetArgs([]string{"stackname"})
	cmd.Flags().Set("compose-file", "-")
	cmd.SetOutput(ioutil.Discard)

	assert.ErrorContains(t, cmd.Execute(), `update out of sequence`)
}
