package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NewLaunchCommand creates a new cobra.Command for `docker app launch`
func NewLaunchCommand(dockerCli command.Cli) *cobra.Command {
	var options *AppOptions

	cmd := &cobra.Command{
		Use:   "launch [OPTIONS] URL [COMMAND] [ARG...]",
		Short: "Launch app from URL",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.setArgs(args)
			adapter := newDockerCliAdapter(dockerCli)
			return runApp(cmd.Context(), adapter, cmd.Flags(), options)
		},
	}

	flags := cmd.Flags()
	flags.SetInterspersed(false)

	id := time.Now().UnixNano()
	dest := filepath.Join(os.TempDir(), fmt.Sprintf("docker-app-launch-%d", id))

	options = addInstallFlags(flags, dest, dockerCli.ContentTrustEnabled())
	flags.Lookup("destination").DefValue = "auto"

	flags.MarkHidden("launch")
	// and more
	markFlagsHiddenExcept(cmd, []string{"destination", "detach", "quiet"}...)

	return cmd
}

func runApp(ctx context.Context, adapter cliAdapter, flags *pflag.FlagSet, options *AppOptions) error {
	if err := validateAppOptions(options); err != nil {
		return err
	}

	dir, err := runInstall(ctx, adapter, flags, options)
	if err != nil {
		return err
	}

	return runLaunch(dir, options)
}

func runLaunch(dir string, options *AppOptions) error {
	locate := func() (string, error) {
		if fp, err := oneChild(dir); err == nil && fp != "" {
			return fp, nil
		}
		appName := options.name
		if appName == "" {
			appName = runnerName
		}
		if fp, err := locateFile(dir, appName); err == nil && fp != "" {
			return fp, nil
		}
		return "", errors.New("no app file found")
	}

	fp, err := locate()
	if err != nil {
		return err
	}

	return launch(fp, options)
}

// launch copies the current environment and set DOCKER_APP_BASE before spawning the app
func launch(app string, options *AppOptions) error {
	envs, err := options.makeEnvs()
	if err != nil {
		return err
	}
	return spawn(app, options.launchArgs(), envs, options.detach)
}
