package swarm

import si "github.com/docker/cli/cli/command/internal/stack/swarm"

var (
	RunDeploy   = si.RunDeploy
	GetStacks   = si.GetStacks
	RunPS       = si.RunPS
	RunRemove   = si.RunRemove
	GetServices = si.GetServices
)

const (
	ResolveImageAlways  = si.ResolveImageAlways
	ResolveImageChanged = si.ResolveImageChanged
	ResolveImageNever   = si.ResolveImageNever
)
