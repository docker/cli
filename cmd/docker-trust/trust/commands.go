package trust

import (
	"fmt"

	"github.com/docker/cli-docs-tool/annotation"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli-plugins/plugin"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/debug"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewRootCmd(name string, isPlugin bool, dockerCLI *command.DockerCli) *cobra.Command {
	var opt rootOptions
	cmd := &cobra.Command{
		Use:   name,
		Short: "Manage trust on Docker images",
		Long:  `Extended build capabilities with BuildKit`,
		Annotations: map[string]string{
			annotation.CodeDelimiter: `"`,
		},
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opt.debug {
				debug.Enable()
			}
			// cmd.SetContext(appcontext.Context())
			if !isPlugin {
				// InstallFlags and SetDefaultOptions are necessary to match
				// the plugin mode behavior to handle env vars such as
				// DOCKER_TLS, DOCKER_TLS_VERIFY, ... and we also need to use a
				// new flagset to avoid conflict with the global debug flag
				// that we already handle in the root command otherwise it
				// would panic.
				nflags := pflag.NewFlagSet(cmd.DisplayName(), pflag.ContinueOnError)
				options := cliflags.NewClientOptions()
				options.InstallFlags(nflags)
				options.SetDefaultOptions(nflags)
				return dockerCLI.Initialize(options)
			}
			return plugin.PersistentPreRunE(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			_ = cmd.Help()
			return cli.StatusError{
				StatusCode: 1,
				Status:     fmt.Sprintf("ERROR: unknown command: %q", args[0]),
			}
		},
	}
	if !isPlugin {
		// match plugin behavior for standalone mode
		// https://github.com/docker/cli/blob/6c9eb708fa6d17765d71965f90e1c59cea686ee9/cli-plugins/plugin/plugin.go#L117-L127
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.TraverseChildren = true
		cmd.DisableFlagsInUseLine = true
	}

	cmd.AddCommand(
		newRevokeCommand(dockerCLI),
		newSignCommand(dockerCLI),
		newTrustKeyCommand(dockerCLI),
		newTrustSignerCommand(dockerCLI),
		newInspectCommand(dockerCLI),
	)

	return cmd
}

type rootOptions struct {
	debug bool
}
