package command

import (
	"os"
	"strings"

	cliconfig "github.com/docker/cli/cli/config"
)

// Orchestrator type acts as an enum describing supported orchestrators.
type Orchestrator string

const (
	// Kubernetes orchestrator
	OrchestratorKubernetes = Orchestrator("kubernetes")
	// Swarm orchestrator
	OrchestratorSwarm = Orchestrator("swarm")
	orchestratorUnset = Orchestrator("unset")

	defaultOrchestrator = OrchestratorSwarm
	dockerOrchestrator  = "DOCKER_ORCHESTRATOR"
)

func normalize(flag string) Orchestrator {
	switch strings.ToLower(flag) {
	case "kubernetes", "k8s":
		return OrchestratorKubernetes
	case "swarm", "swarmkit":
		return OrchestratorSwarm
	default:
		return orchestratorUnset
	}
}

// GetOrchestrator checks DOCKER_ORCHESTRATOR environment variable and configuration file
// orchestrator value and returns user defined Orchestrator.
func GetOrchestrator(dockerCli Cli) Orchestrator {
	// Check environment variable
	env := os.Getenv(dockerOrchestrator)
	if o := normalize(env); o != orchestratorUnset {
		return o
	}
	// Check config file
	if configFile := cliconfig.LoadDefaultConfigFile(dockerCli.Err()); configFile != nil {
		if o := normalize(configFile.Orchestrator); o != orchestratorUnset {
			return o
		}
	}

	// Nothing set, use default orchestrator
	return defaultOrchestrator
}
