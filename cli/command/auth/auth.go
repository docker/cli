package auth

import (
	"fmt"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/server"
	"github.com/spf13/cobra"
)

func NewAuthCommand() *cobra.Command {
	authCmd := &cobra.Command{
		Use: "auth",
	}

	proxyServerCmd := &cobra.Command{
		Use: "credential-server",
		RunE: func(cmd *cobra.Command, args []string) error {
			file := config.LoadDefaultConfigFile(cmd.ErrOrStderr())
			fmt.Fprint(cmd.OutOrStdout(), "Starting credential server...\n")
			err := server.StartCredentialsServer(cmd.Context(), config.Dir(), file)
			if err != nil {
				return err
			}
			return nil
		},
	}
	authCmd.AddCommand(proxyServerCmd)

	return authCmd
}
