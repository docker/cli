package volume

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type renameOptions struct {
	volume    string
	newVolume string
}

func newRenameCommand(dockerCli command.Cli) *cobra.Command {
	var opts renameOptions

	cmd := &cobra.Command{
		Use:     "rename VOLUME NEW_NAME",
		Aliases: []string{"mv"},
		Short:   "Rename a volume",
		Long:    renameDescription,
		Example: renameExample,
		Args:    cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.volume = args[0]
			opts.newVolume = args[1]
			return runRename(dockerCli, &opts)
		},
	}
	return cmd
}

func runRename(dockerCli command.Cli, opts *renameOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	if err := client.VolumeRename(ctx, opts.volume, opts.newVolume); err != nil {
		return err
	}
	fmt.Fprintf(dockerCli.Out(), "%s\n", opts.newVolume)
	return nil
}

var renameDescription = `
Rename a volume.
`

var renameExample = `
$ docker volume rename docker moby
moby
`
