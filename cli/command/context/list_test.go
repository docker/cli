package context

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
)

func createTestContexts(t *testing.T, cli command.Cli, name ...string) {
	t.Helper()
	for _, n := range name {
		createTestContext(t, cli, n, nil)
	}
}

func createTestContext(t *testing.T, cli command.Cli, name string, metaData map[string]any) {
	t.Helper()

	err := RunCreate(cli, &CreateOptions{
		Name:        name,
		Description: "description of " + name,
		Docker:      map[string]string{keyHost: "https://someswarmserver.example.com"},

		metaData: metaData,
	})
	assert.NilError(t, err)
}

func TestList(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContexts(t, cli, "current", "other", "unset")
	cli.SetCurrentContext("current")
	cli.OutBuffer().Reset()
	assert.NilError(t, runList(cli, &listOptions{}))
	golden.Assert(t, cli.OutBuffer().String(), "list.golden")
}

func TestListQuiet(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContexts(t, cli, "current", "other")
	cli.SetCurrentContext("current")
	cli.OutBuffer().Reset()
	assert.NilError(t, runList(cli, &listOptions{quiet: true}))
	golden.Assert(t, cli.OutBuffer().String(), "quiet-list.golden")
}

func TestListError(t *testing.T) {
	cli := makeFakeCli(t)
	cli.SetCurrentContext("nosuchcontext")
	cli.OutBuffer().Reset()
	assert.NilError(t, runList(cli, &listOptions{}))
	golden.Assert(t, cli.OutBuffer().String(), "list-with-error.golden")
}
