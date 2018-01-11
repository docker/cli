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
	orchestratorUnset = Orchestrator("unset")

	defaultOrchestrator      = OrchestratorSwarm
	envVarDockerOrchestrator = "DOCKER_ORCHESTRATOR"
)

func normalize(flag string) Orchestrator {
	switch flag {
	case "kubernetes":
		return OrchestratorKubernetes
	case "swarm":
		return OrchestratorSwarm
	default:
		return orchestratorUnset
	}
}

// GetOrchestrator checks DOCKER_ORCHESTRATOR environment variable and configuration file
// orchestrator value and returns user defined Orchestrator.
func GetOrchestrator(isExperimental bool, flagValue, value string) Orchestrator {
	// Non experimental CLI has kubernetes disabled
	if !isExperimental {
		return defaultOrchestrator
	}
	// Check flag
	if o := normalize(flagValue); o != orchestratorUnset {
		return o
	}
	// Check environment variable
	env := os.Getenv(envVarDockerOrchestrator)
	if o := normalize(env); o != orchestratorUnset {
		return o
	}
	// Check specified orchestrator
	if o := normalize(value); o != orchestratorUnset {
		return o
	}

	if value != "" {
		fmt.Fprintf(os.Stderr, "Specified orchestrator %q is invalid. Please use either kubernetes or swarm\n", value)
	}
	// Nothing set, use default orchestrator
	return defaultOrchestrator
}
