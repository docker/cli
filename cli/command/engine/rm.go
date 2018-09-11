package engine

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// TODO - consider adding a "purge" flag that also removes
// configuration files and the docker root dir.

type rmOptions struct {
	sockPath string
}

func newRmCommand(dockerCli command.Cli) *cobra.Command {
	var options rmOptions
	cmd := &cobra.Command{
		Use:   "rm [OPTIONS]",
		Short: "Remove the local engine",
		Long: `This command will remove the local engine running on containerd.

No state files will be removed from the host filesystem.
`,
		Args: cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(dockerCli, options)
		},
		Annotations: map[string]string{"experimentalCLI": ""},
	}
	flags := cmd.Flags()
	flags.StringVar(&options.sockPath, "containerd", "", "override default location of containerd endpoint")

	return cmd
}

func runRm(dockerCli command.Cli, options rmOptions) error {
	ctx := context.Background()
	client, err := dockerCli.NewContainerizedEngineClient(options.sockPath)
	if err != nil {
		return errors.Wrap(err, "unable to access local containerd")
	}
	defer client.Close()

	return client.RemoveEngine(ctx)
}
