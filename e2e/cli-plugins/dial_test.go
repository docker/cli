package cliplugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli-plugins/manager"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/icmd"
)

func TestCLIPluginDialStdio(t *testing.T) {
	if os.Getenv("DOCKER_CLI_PLUGIN_USE_DIAL_STDIO") == "" {
		t.Skip("skipping plugin dial-stdio test since DOCKER_CLI_PLUGIN_USE_DIAL_STDIO is not set")
	}

	// Run the helloworld plugin forcing /bin/true as the `system
	// dial-stdio` target. It should be passed all arguments from
	// before the `helloworld` arg, but not the --who=foo which
	// follows. We observe this from the debug level logging from
	// the connhelper stuff.
	helloworld := filepath.Join(os.Getenv("DOCKER_CLI_E2E_PLUGINS_EXTRA_DIRS"), "docker-helloworld")
	cmd := icmd.Command(helloworld, "--config=blah", "--log-level", "debug", "helloworld", "--who=foo")
	res := icmd.RunCmd(cmd, icmd.WithEnv(manager.ReexecEnvvar+"=/bin/true"))
	res.Assert(t, icmd.Success)
	assert.Assert(t, is.Contains(res.Stderr(), `msg="commandconn: starting /bin/true with [--config=blah --log-level debug system dial-stdio]"`))
	assert.Assert(t, is.Equal(res.Stdout(), "Hello foo!\n"))
}
