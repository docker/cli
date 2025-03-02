package context

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestExportImportWithFile(t *testing.T) {
	contextFile := filepath.Join(t.TempDir(), "exported")
	cli := makeFakeCli(t)
	createTestContext(t, cli, "test", map[string]any{
		"MyCustomMetadata": t.Name(),
	})
	cli.ErrBuffer().Reset()
	assert.NilError(t, RunExport(cli, &ExportOptions{
		ContextName: "test",
		Dest:        contextFile,
	}))
	assert.Equal(t, cli.ErrBuffer().String(), fmt.Sprintf("Written file %q\n", contextFile))
	cli.OutBuffer().Reset()
	cli.ErrBuffer().Reset()
	assert.NilError(t, RunImport(cli, "test2", contextFile))
	context1, err := cli.ContextStore().GetMetadata("test")
	assert.NilError(t, err)
	context2, err := cli.ContextStore().GetMetadata("test2")
	assert.NilError(t, err)

	assert.Check(t, is.DeepEqual(context1.Metadata, command.DockerContext{
		Description:      "description of test",
		AdditionalFields: map[string]any{"MyCustomMetadata": t.Name()},
	}))

	assert.Check(t, is.DeepEqual(context1.Endpoints, context2.Endpoints))
	assert.Check(t, is.DeepEqual(context1.Metadata, context2.Metadata))
	assert.Check(t, is.Equal("test", context1.Name))
	assert.Check(t, is.Equal("test2", context2.Name))

	assert.Check(t, is.Equal("test2\n", cli.OutBuffer().String()))
	assert.Check(t, is.Equal("Successfully imported context \"test2\"\n", cli.ErrBuffer().String()))
}

func TestExportImportPipe(t *testing.T) {
	cli := makeFakeCli(t)
	createTestContext(t, cli, "test", map[string]any{
		"MyCustomMetadata": t.Name(),
	})
	cli.ErrBuffer().Reset()
	cli.OutBuffer().Reset()
	assert.NilError(t, RunExport(cli, &ExportOptions{
		ContextName: "test",
		Dest:        "-",
	}))
	assert.Equal(t, cli.ErrBuffer().String(), "")
	cli.SetIn(streams.NewIn(io.NopCloser(bytes.NewBuffer(cli.OutBuffer().Bytes()))))
	cli.OutBuffer().Reset()
	cli.ErrBuffer().Reset()
	assert.NilError(t, RunImport(cli, "test2", "-"))
	context1, err := cli.ContextStore().GetMetadata("test")
	assert.NilError(t, err)
	context2, err := cli.ContextStore().GetMetadata("test2")
	assert.NilError(t, err)

	assert.Check(t, is.DeepEqual(context1.Metadata, command.DockerContext{
		Description:      "description of test",
		AdditionalFields: map[string]any{"MyCustomMetadata": t.Name()},
	}))

	assert.Check(t, is.DeepEqual(context1.Endpoints, context2.Endpoints))
	assert.Check(t, is.DeepEqual(context1.Metadata, context2.Metadata))
	assert.Check(t, is.Equal("test", context1.Name))
	assert.Check(t, is.Equal("test2", context2.Name))

	assert.Check(t, is.Equal("test2\n", cli.OutBuffer().String()))
	assert.Check(t, is.Equal("Successfully imported context \"test2\"\n", cli.ErrBuffer().String()))
}

func TestExportExistingFile(t *testing.T) {
	contextFile := filepath.Join(t.TempDir(), "exported")
	cli := makeFakeCli(t)
	cli.ErrBuffer().Reset()
	assert.NilError(t, os.WriteFile(contextFile, []byte{}, 0o644))
	err := RunExport(cli, &ExportOptions{ContextName: "test", Dest: contextFile})
	assert.Assert(t, os.IsExist(err))
}
