package context

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
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

func TestListJSON(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "current", nil)
	createTestContext(t, cli, "context1", map[string]any{"Type": "aci"})
	createTestContext(t, cli, "context2", map[string]any{"Type": "ecs"})
	createTestContext(t, cli, "context3", map[string]any{"Type": "moby"})
	cli.SetCurrentContext("current")

	t.Run("format={{json .}}", func(t *testing.T) {
		cli.OutBuffer().Reset()
		assert.NilError(t, runList(cli, &listOptions{format: formatter.JSONFormat}))
		golden.Assert(t, cli.OutBuffer().String(), "list-json.golden")
	})

	t.Run("format=json", func(t *testing.T) {
		cli.OutBuffer().Reset()
		assert.NilError(t, runList(cli, &listOptions{format: formatter.JSONFormatKey}))
		golden.Assert(t, cli.OutBuffer().String(), "list-json.golden")
	})

	t.Run("format={{ json .Name }}", func(t *testing.T) {
		cli.OutBuffer().Reset()
		assert.NilError(t, runList(cli, &listOptions{format: `{{ json .Name }}`}))
		golden.Assert(t, cli.OutBuffer().String(), "list-json-name.golden")
	})
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
