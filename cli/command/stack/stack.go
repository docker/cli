package stack

import si "github.com/docker/cli/cli/command/internal/stack"

// orchestration (swarm) commands
var (
	NewStackCommand = si.NewStackCommand
	RunList         = si.RunList
	RunServices     = si.RunServices
)
