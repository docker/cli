package context

import (
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/golden"
)

func TestInspect(t *testing.T) {
	cli, cleanup := makeFakeCli(t)
	defer cleanup()
	createTestContextWithKubeAndSwarm(t, cli, "current", "all")
	cli.OutBuffer().Reset()
	assert.NilError(t, runInspect(cli, inspectOptions{
		refs: []string{"current"},
	}))
	expected := string(golden.Get(t, "inspect.golden"))
	si := cli.ContextStore().GetContextStorageInfo("current")
	expected = strings.Replace(expected, "<METADATA_PATH>", strings.Replace(si.MetadataPath, `\`, `\\`, -1), 1)
	expected = strings.Replace(expected, "<TLS_PATH>", strings.Replace(si.TLSPath, `\`, `\\`, -1), 1)
	assert.Equal(t, cli.OutBuffer().String(), expected)
}
