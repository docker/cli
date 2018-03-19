package engine

import (
	"context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/containerizedengine"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type extendedEngineInitOptions struct {
	containerizedengine.EngineInitOptions
	sockPath string
}

func newInitCommand(dockerCli command.Cli) *cobra.Command {
	var options extendedEngineInitOptions

	cmd := &cobra.Command{
		Use:   "init [OPTIONS]",
		Short: "Initialize a local engine",
		Long: `This command will initialize a local engine running on containerd.

Configuration of the engine is managed through the daemon.json configuration
file on the host and may be pre-created before running the 'init' command.
`,
		Args: cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(dockerCli, options)
		},
		Annotations: map[string]string{"experimentalCLI": ""},
	}
	flags := cmd.Flags()
	flags.StringVar(&options.EngineVersion, "version", cli.Version, "Specify engine version")
	flags.StringVar(&options.EngineImage, "engine-image", containerizedengine.CommunityEngineImage, "Specify engine image")
	flags.StringVar(&options.RegistryPrefix, "registry-prefix", "docker.io/docker", "Override the default location where engine images are pulled")
	flags.StringVar(&options.ConfigFile, "config-file", "/etc/docker/daemon.json", "Specify the location of the daemon configuration file on the host")
	flags.StringVar(&options.sockPath, "containerd", "", "override default location of containerd endpoint")

	return cmd
}

func runInit(dockerCli command.Cli, options extendedEngineInitOptions) error {
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
	return client.InitEngine(ctx, options.EngineInitOptions, dockerCli.Out(), authConfig,
		func(ctx context.Context) error {
			client := dockerCli.Client()
			_, err := client.Ping(ctx)
			return err
		})
}
