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

			switch args[0] {
			case "install":

				userHome, err := os.UserHomeDir()
				if err != nil {
					return err
				}

				opts := []NewShellCompletionOptsFunc{}
				if cmd.Flag("shell").Changed {
					opts = append(opts, WithShellOverride(cmd.Flag("shell").Value.String()))
				}

				shellSetup, err := NewShellCompletionSetup(userHome, cmd.Root(), opts...)
				if err != nil {
					return err
				}

				if cmd.Flag("manual").Changed {
					_, _ = fmt.Fprint(dockerCli.Out(), shellSetup.GetManualInstructions(cmd.Context()))
					return nil
				}

				msg := fmt.Sprintf("\nDetected shell [%s]\n\nThe automatic installer will do the following:\n\n%s\n\nAre you sure you want to continue?", shellSetup.GetShell(), shellSetup.GetManualInstructions(cmd.Context()))
				ok, err := command.PromptForConfirmation(cmd.Context(), dockerCli.In(), dockerCli.Out(), msg)
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}

				return shellSetup.InstallCompletions(cmd.Context())
			case "bash":

				return cmd.Root().GenBashCompletionV2(dockerCli.Out(), true)
			case "zsh":
				return cmd.Root().GenZshCompletion(dockerCli.Out())
			case "fish":
				return cmd.Root().GenFishCompletion(dockerCli.Out(), true)
			default:
				return command.ShowHelp(dockerCli.Err())(cmd, args)
			}
		},
	}

	cmd.PersistentFlags().Bool("manual", false, "Display instructions for installing autocompletion")
	cmd.PersistentFlags().String("shell", "", "Shell type for autocompletion (bash, zsh, fish, powershell)")

	return cmd
}
