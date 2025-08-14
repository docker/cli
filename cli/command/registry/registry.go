package registry

import ri "github.com/docker/cli/cli/command/internal/registry"

var (
	NewLoginCommand  = ri.NewLoginCommand
	NewLogoutCommand = ri.NewLogoutCommand
	NewSearchCommand = ri.NewSearchCommand
)
