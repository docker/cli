package engine

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	clitypes "github.com/docker/cli/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newUpdateCommand(dockerCli command.Cli) *cobra.Command {
	var options extendedEngineInitOptions

	cmd := &cobra.Command{
		Use:   "update [OPTIONS]",
		Short: "Update a local engine",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(dockerCli, options)
		},
	}
	flags := cmd.Flags()

	flags.StringVar(&options.EngineVersion, "version", "", "Specify engine version")
	flags.StringVar(&options.EngineImage, "engine-image", "", "Specify engine image")
	flags.StringVar(&options.RegistryPrefix, "registry-prefix", clitypes.RegistryPrefix, "Override the current location where engine images are pulled")
	flags.StringVar(&options.sockPath, "containerd", "", "override default location of containerd endpoint")

	return cmd
}

func runUpdate(dockerCli command.Cli, options extendedEngineInitOptions) error {
	if !isRoot() {
		return errors.New("this command must be run as a privileged user")
	}
	ctx := context.Background()
	client, err := dockerCli.NewContainerizedEngineClient(options.sockPath)
	if err != nil {
		return errors.Wrap(err, "unable to access local containerd")
	}
	defer client.Close()
	authConfig, err := getRegistryAuth(dockerCli, options.RegistryPrefix)
	if err != nil {
		return err
	}

	if err := client.DoUpdate(ctx, options.EngineInitOptions, dockerCli.Out(), authConfig,
		func(ctx context.Context) error {
			client := dockerCli.Client()
			_, err := client.Ping(ctx)
			return err
		}); err != nil {
		return err
	}
	fmt.Fprintln(dockerCli.Out(), `Succesfully updated engine.
Restart docker with 'systemctl restart docker' to complete the update.`)
	return nil
}
