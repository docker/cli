package kubernetes

import (
	"os"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/gotestyourself/gotestyourself/icmd"
)

// WithOrchestrator updates the specified config to enable kubernetes
func WithOrchestrator(config *configfile.ConfigFile) {
	config.Orchestrator = "kubernetes"
}

// WithKubeConfig sets KUBECONFIG environment variable to the command
func WithKubeConfig(path string) func(cmd *icmd.Cmd) {
	return func(cmd *icmd.Cmd) {
		env := append(os.Environ(),
			"KUBECONFIG="+path,
		)
		cmd.Env = append(cmd.Env, env...)
	}
}
