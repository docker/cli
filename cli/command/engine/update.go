package engine

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
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
	flags.StringVar(&options.RegistryPrefix, "registry-prefix", "", "Override the current location where engine images are pulled")
	flags.StringVar(&options.sockPath, "containerd", "", "override default location of containerd endpoint")

	return cmd
}

func runUpdate(dockerCli command.Cli, options extendedEngineInitOptions) error {
	ctx := context.Background()
	client, err := dockerCli.NewContainerizedEngineClient(options.sockPath)
	if err != nil {
		return errors.Wrap(err, "unable to access local containerd")
	}
	defer client.Close()
	if options.EngineImage == "" || options.RegistryPrefix == "" {
		currentOpts, err := client.GetCurrentEngineVersion(ctx)
		if err != nil {
			return err
		}
		if options.EngineImage == "" {
			options.EngineImage = currentOpts.EngineImage
		}
		if options.RegistryPrefix == "" {
			options.RegistryPrefix = currentOpts.RegistryPrefix
		}
	}
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
	fmt.Fprintln(dockerCli.Out(), "Success!  The docker engine is now running.")
	return nil
}
