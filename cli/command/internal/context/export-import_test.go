package context

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/streams"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestExportImportWithFile(t *testing.T) {
	contextFile := filepath.Join(t.TempDir(), "exported")
	fakeCli := makeFakeCli(t)
	createTestContext(t, fakeCli, "test", map[string]any{
		"MyCustomMetadata": t.Name(),
	})
	fakeCli.ErrBuffer().Reset()
	assert.NilError(t, RunExport(fakeCli, &ExportOptions{
		ContextName: "test",
		Dest:        contextFile,
	}))
	assert.Equal(t, fakeCli.ErrBuffer().String(), fmt.Sprintf("Written file %q\n", contextFile))
	fakeCli.OutBuffer().Reset()
	fakeCli.ErrBuffer().Reset()
	assert.NilError(t, RunImport(fakeCli, "test2", contextFile))
	context1, err := fakeCli.ContextStore().GetMetadata("test")
	assert.NilError(t, err)
	context2, err := fakeCli.ContextStore().GetMetadata("test2")
	assert.NilError(t, err)

	assert.Check(t, is.DeepEqual(context1.Metadata, cli.DockerContext{
		Description:      "description of test",
		AdditionalFields: map[string]any{"MyCustomMetadata": t.Name()},
	}))

	assert.Check(t, is.DeepEqual(context1.Endpoints, context2.Endpoints))
	assert.Check(t, is.DeepEqual(context1.Metadata, context2.Metadata))
	assert.Check(t, is.Equal("test", context1.Name))
	assert.Check(t, is.Equal("test2", context2.Name))

	assert.Check(t, is.Equal("test2\n", fakeCli.OutBuffer().String()))
	assert.Check(t, is.Equal("Successfully imported context \"test2\"\n", fakeCli.ErrBuffer().String()))
}

func TestExportImportPipe(t *testing.T) {
	fakeCli := makeFakeCli(t)
	createTestContext(t, fakeCli, "test", map[string]any{
		"MyCustomMetadata": t.Name(),
	})
	fakeCli.ErrBuffer().Reset()
	fakeCli.OutBuffer().Reset()
	assert.NilError(t, RunExport(fakeCli, &ExportOptions{
		ContextName: "test",
		Dest:        "-",
	}))
	assert.Equal(t, fakeCli.ErrBuffer().String(), "")
	fakeCli.SetIn(streams.NewIn(io.NopCloser(bytes.NewBuffer(fakeCli.OutBuffer().Bytes()))))
	fakeCli.OutBuffer().Reset()
	fakeCli.ErrBuffer().Reset()
	assert.NilError(t, RunImport(fakeCli, "test2", "-"))
	context1, err := fakeCli.ContextStore().GetMetadata("test")
	assert.NilError(t, err)
	context2, err := fakeCli.ContextStore().GetMetadata("test2")
	assert.NilError(t, err)

	assert.Check(t, is.DeepEqual(context1.Metadata, cli.DockerContext{
		Description:      "description of test",
		AdditionalFields: map[string]any{"MyCustomMetadata": t.Name()},
	}))

	assert.Check(t, is.DeepEqual(context1.Endpoints, context2.Endpoints))
	assert.Check(t, is.DeepEqual(context1.Metadata, context2.Metadata))
	assert.Check(t, is.Equal("test", context1.Name))
	assert.Check(t, is.Equal("test2", context2.Name))

	assert.Check(t, is.Equal("test2\n", fakeCli.OutBuffer().String()))
	assert.Check(t, is.Equal("Successfully imported context \"test2\"\n", fakeCli.ErrBuffer().String()))
}

func TestExportExistingFile(t *testing.T) {
	contextFile := filepath.Join(t.TempDir(), "exported")
	fakeCli := makeFakeCli(t)
	fakeCli.ErrBuffer().Reset()
	assert.NilError(t, os.WriteFile(contextFile, []byte{}, 0o644))
	err := RunExport(fakeCli, &ExportOptions{ContextName: "test", Dest: contextFile})
	assert.Assert(t, os.IsExist(err))
}
