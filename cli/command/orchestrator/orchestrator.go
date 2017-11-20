package orchestrator

import (
	"os"
	"strings"

	"github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
)

// Orchestrator type acts as an enum describing supported orchestrators.
type Orchestrator string

const (
	// Kubernetes orchestrator
	Kubernetes = Orchestrator("kubernetes")
	// Swarm orchestrator
	Swarm = Orchestrator("swarm")
	unset = Orchestrator("unset")

	defaultOrchestrator = Swarm
	dockerOrchestrator  = "DOCKER_ORCHESTRATOR"
)

func normalize(flag string) Orchestrator {
	switch strings.ToLower(flag) {
	case "kubernetes", "k8s":
		return Kubernetes
	case "swarm", "swarmkit":
		return Swarm
	default:
		return unset
	}
}

// GetOrchestrator checks DOCKER_ORCHESTRATOR environment variable and configuration file
// orchestrator value and returns user defined Orchestrator.
func GetOrchestrator(dockerCli command.Cli) Orchestrator {
	// Check environment variable
	env := os.Getenv(dockerOrchestrator)
	if o := normalize(env); o != unset {
		return o
	}
	// Check config file
	if configFile := cliconfig.LoadDefaultConfigFile(dockerCli.Err()); configFile != nil {
		if o := normalize(configFile.Orchestrator); o != unset {
			return o
		}
	}

	// Nothing set, use default orchestrator
	return defaultOrchestrator
}
