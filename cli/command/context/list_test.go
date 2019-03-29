package context

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"gotest.tools/assert"
	"gotest.tools/env"
	"gotest.tools/golden"
)

func createTestContextWithKubeAndSwarm(t *testing.T, cli command.Cli, name string, orchestrator string) {
	revert := env.Patch(t, "KUBECONFIG", "./testdata/test-kubeconfig")
	defer revert()

	err := RunCreate(cli, &CreateOptions{
		Name:                     name,
		DefaultStackOrchestrator: orchestrator,
		Description:              "description of " + name,
		Kubernetes:               map[string]string{keyFromCurrent: "true"},
		Docker:                   map[string]string{keyHost: "https://someswarmserver"},
	})
	assert.NilError(t, err)
}

func TestList(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "current", "all")
	createTestContextWithKubeAndSwarm(t, cli, "other", "all")
	createTestContextWithKubeAndSwarm(t, cli, "unset", "unset")
	cli.SetCurrentContext("current")
	cli.OutBuffer().Reset()
	assert.NilError(t, runList(cli, &listOptions{}))
	golden.Assert(t, cli.OutBuffer().String(), "list.golden")
}

func TestListQuiet(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "current", "all")
	createTestContextWithKubeAndSwarm(t, cli, "other", "all")
	cli.SetCurrentContext("current")
	cli.OutBuffer().Reset()
	assert.NilError(t, runList(cli, &listOptions{quiet: true}))
	golden.Assert(t, cli.OutBuffer().String(), "quiet-list.golden")
}
