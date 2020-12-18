package compose

import (
	"github.com/docker/cli/backends"
	"github.com/spf13/cobra"
)

// NewComposeCommand compose command delegated to backend
func NewComposeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "compose",
		Short:              "Manage compose projects",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return backends.RunBackendCLI(backends.ContextTypeLocal)
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use: "fake command so compose is a Management Command",
	})
	return cmd
}
