package serve

import (
	"github.com/docker/cli/backends"
	"github.com/spf13/cobra"
)

// NewServeCommand command to serve gRPC API for backend commands (ACI/ECS for now) delegated to backend
func NewServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "serve",
		Short:              "Start a Docker client api server",
		DisableFlagParsing: true,
		Hidden:             true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return backends.RunBackendCLI(backends.ContextTypeLocal)
		},
	}
	return cmd
}
