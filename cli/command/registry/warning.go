package registry

import (
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/internal/tui"
)

// maybePrintEnvAuthWarning if the `DOCKER_AUTH_CONFIG` environment variable is
// set this function will output a warning to stdErr
func maybePrintEnvAuthWarning(out command.Streams) {
	if os.Getenv(configfile.DockerEnvConfigKey) != "" {
		tui.NewOutput(out.Err()).
			PrintWarning("%[1]s is set and takes precedence.\nUnset %[1]s to restore the CLI auth behaviour.\n", configfile.DockerEnvConfigKey)
	}
}
