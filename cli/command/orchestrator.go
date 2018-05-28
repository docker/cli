package command

import (
	"fmt"
	"os"
)

// Orchestrator type acts as an enum describing supported orchestrators.
type Orchestrator string

const (
	// OrchestratorKubernetes orchestrator
	OrchestratorKubernetes = Orchestrator("kubernetes")
	// OrchestratorSwarm orchestrator
	OrchestratorSwarm = Orchestrator("swarm")
	// OrchestratorAll orchestrator
	OrchestratorAll   = Orchestrator("all")
	orchestratorUnset = Orchestrator("unset")

	defaultOrchestrator      = OrchestratorSwarm
	envVarDockerOrchestrator = "DOCKER_ORCHESTRATOR"
)

func normalize(value string) (Orchestrator, error) {
	switch value {
	case "kubernetes":
		return OrchestratorKubernetes, nil
	case "swarm":
		return OrchestratorSwarm, nil
	case "":
		return orchestratorUnset, nil
	case "all":
		return OrchestratorAll, nil
	default:
		return defaultOrchestrator, fmt.Errorf("specified orchestrator %q is invalid, please use either kubernetes, swarm or all", value)
	}
}

// GetOrchestrator checks DOCKER_ORCHESTRATOR environment variable and configuration file
// orchestrator value and returns user defined Orchestrator.
func GetOrchestrator(flagValue, value string) (Orchestrator, error) {
	// Check flag
	if o, err := normalize(flagValue); o != orchestratorUnset {
		return o, err
	}
	// Check environment variable
	env := os.Getenv(envVarDockerOrchestrator)
	if o, err := normalize(env); o != orchestratorUnset {
		return o, err
	}
	// Check specified orchestrator
	if o, err := normalize(value); o != orchestratorUnset {
		return o, err
	}
	// Nothing set, use default orchestrator
	return defaultOrchestrator, nil
}
