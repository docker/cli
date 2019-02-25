package cliplugins

import (
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli/config"
	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestConfig(t *testing.T) {
	run, cfg, cleanup := prepare(t)
	defer cleanup()

	cfg.SetPluginConfig("helloworld", "who", "Cambridge")
	err := cfg.Save()
	assert.NilError(t, err)

	res := icmd.RunCmd(run("helloworld"))
	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Out:      "Hello Cambridge!",
	})

	cfg2, err := config.Load(filepath.Dir(cfg.GetFilename()))
	assert.NilError(t, err)
	assert.DeepEqual(t, cfg2.Plugins, map[string]map[string]string{
		"helloworld": {
			"who":     "Cambridge",
			"lastwho": "Cambridge",
		},
	})
}
