package completion

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func NewCompletionCommand(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "completion [bash|zsh|fish|powershell|install]",
		Short:              "Output shell completion code for the specified shell (bash or zsh)",
		Args:               cli.RequiresMaxArgs(1),
		ValidArgs:          []string{"bash", "zsh", "fish", "powershell", "install"},
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			shellSetup := NewUnixShellSetup("", "docker")

			if cmd.Flag("manual").Changed {
				_, _ = fmt.Fprint(dockerCli.Out(), shellSetup.GetManualInstructions(supportedCompletionShell(args[0])))
				return nil
			}

			switch args[0] {
			case "install":
				return shellSetup.InstallCompletions(cmd.Context(), supportedCompletionShell(os.Getenv("SHELL")))
			case "bash":

				return cmd.GenBashCompletionV2(dockerCli.Out(), true)
			case "zsh":
				return cmd.GenZshCompletion(dockerCli.Out())
			case "fish":
				return cmd.GenFishCompletion(dockerCli.Out(), true)
			default:
				return command.ShowHelp(dockerCli.Err())(cmd, args)
			}
		},
	}

	cmd.PersistentFlags().Bool("manual", false, "Display instructions for installing autocompletion")

	return cmd
}
