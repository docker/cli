package job

import (
	"testing"

	"github.com/docker/cli/internal/test"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

// The jobs extension shipped with Engine API 1.57: the CLI hides the group and
// refuses it with a "requires API version" error on older daemons, instead of
// suggesting an --extension-enable flag they do not have.
func TestNewJobCommandRequiresAPIVersion(t *testing.T) {
	cmd := newJobCommand(test.NewFakeCli(nil))

	assert.Check(t, is.Equal(cmd.Annotations["version"], "1.57"))
	assert.Check(t, is.Contains(cmd.Long, "--extension-enable org.mobyproject.jobs.v0"))
}
