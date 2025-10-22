package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/moby/go-archive"
	"github.com/moby/go-archive/compression"
	"github.com/moby/moby/api/types/plugin"
	"github.com/moby/moby/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// validateTag checks if the given repoName can be resolved.
func validateTag(rawRepo string) error {
	_, err := reference.ParseNormalizedNamed(rawRepo)

	return err
}

// validateConfig ensures that a valid config.json is available in the given path
func validateConfig(path string) error {
	dt, err := os.Open(filepath.Join(path, "config.json"))
	if err != nil {
		return err
	}

	m := plugin.Config{}
	err = json.NewDecoder(dt).Decode(&m)
	_ = dt.Close()

	return err
}

// validateContextDir validates the given dir and returns its absolute path on success.
func validateContextDir(contextDir string) (string, error) {
	absContextDir, err := filepath.Abs(contextDir)
	if err != nil {
		return "", err
	}
	stat, err := os.Lstat(absContextDir)
	if err != nil {
		return "", err
	}

	if !stat.IsDir() {
		return "", errors.New("context must be a directory")
	}

	return absContextDir, nil
}

type pluginCreateOptions struct {
	repoName string
	context  string
	compress bool
}

func newCreateCommand(dockerCLI command.Cli) *cobra.Command {
	options := pluginCreateOptions{}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] PLUGIN PLUGIN-DATA-DIR",
		Short: "Create a plugin from a rootfs and configuration. Plugin data directory must contain config.json and rootfs directory.",
		Args:  cli.RequiresMinArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.repoName = args[0]
			options.context = args[1]
			return runCreate(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction:     cobra.NoFileCompletions, // TODO(thaJeztah): should provide "directory" completion for the second arg
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()

	flags.BoolVar(&options.compress, "compress", false, "Compress the context using gzip")

	return cmd
}

func runCreate(ctx context.Context, dockerCli command.Cli, options pluginCreateOptions) error {
	if err := validateTag(options.repoName); err != nil {
		return err
	}

	absContextDir, err := validateContextDir(options.context)
	if err != nil {
		return err
	}

	if err := validateConfig(options.context); err != nil {
		return err
	}

	comp := compression.None
	if options.compress {
		logrus.Debugf("compression enabled")
		comp = compression.Gzip
	}

	createCtx, err := archive.TarWithOptions(absContextDir, &archive.TarOptions{
		Compression: comp,
	})
	if err != nil {
		return err
	}

	_, err = dockerCli.Client().PluginCreate(ctx, createCtx, client.PluginCreateOptions{RepoName: options.repoName})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(dockerCli.Out(), options.repoName)
	return nil
}
