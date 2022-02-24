package context

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func createTestContext(t *testing.T, cli command.Cli, name string) {
	t.Helper()

	err := RunCreate(cli, &CreateOptions{
		Name:        name,
		Description: "description of " + name,
		Docker:      map[string]string{keyHost: "https://someswarmserver.example.com"},
	})
	assert.NilError(t, err)
}

func TestList(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContext(t, cli, "current")
	createTestContext(t, cli, "other")
	createTestContext(t, cli, "unset")
	cli.SetCurrentContext("current")
	cli.OutBuffer().Reset()
	assert.NilError(t, runList(cli, &listOptions{}))
	golden.Assert(t, cli.OutBuffer().String(), "list.golden")
}

func TestListQuiet(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContext(t, cli, "current")
	createTestContext(t, cli, "other")
	cli.SetCurrentContext("current")
	cli.OutBuffer().Reset()
	assert.NilError(t, runList(cli, &listOptions{quiet: true}))
	golden.Assert(t, cli.OutBuffer().String(), "quiet-list.golden")
}
