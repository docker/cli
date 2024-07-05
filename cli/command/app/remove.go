package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

// NewRemoveCommand creates a new `docker app remove` command
func NewRemoveCommand(dockerCli command.Cli) *cobra.Command {
	var options *removeOptions

	cmd := &cobra.Command{
		Use:     "remove [OPTIONS] URL [URL...]",
		Aliases: []string{"rm", "uninstall"},
		Short:   "Remove one or more applications",
		Args:    cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(dockerCli, args, options)
		},
		Annotations: map[string]string{
			"aliases": "docker app rm, docker app uninstall",
		},
	}

	options = newRemoveOptions()

	return cmd
}

// runRemove removes the specified apps installed under the default app base.
// run uninstall script if found under the package path
// remove all files under the package path
// remove symlinks of the app under bin path
func runRemove(dockerCli command.Cli, apps []string, options *removeOptions) error {
	binPath := options.binPath()

	// find all symlinks in binPath to remove later
	// if they point to the app package
	targets := make(map[string]string)
	if links, err := findLinks(binPath); err == nil {
		for _, link := range links {
			if target, err := os.Readlink(link); err == nil {
				targets[target] = link
			}
		}
	}

	var failed []string

	for _, app := range apps {
		appPath, err := options.makeAppPath(app)
		if err != nil {
			failed = append(failed, app)
			continue
		}

		// optionally run uninstall if provided
		uninstaller := filepath.Join(appPath, uninstallerName)
		if _, err := os.Stat(uninstaller); err == nil {
			err := spawn(uninstaller, nil, nil, false)
			if err != nil {
				fmt.Fprintf(dockerCli.Err(), "%s failed to run: %v\n", uninstaller, err)
			}
		}

		// remove all files under the app path
		if err := os.RemoveAll(appPath); err != nil {
			failed = append(failed, app)
			continue
		}
		removeEmptyPath(options.pkgPath(), appPath)
		fmt.Fprintf(dockerCli.Out(), "app package removed %s\n", appPath)

		// remove symlinks of the app if any
		for target, link := range targets {
			if strings.Contains(target, appPath) {
				if err := os.Remove(link); err != nil {
					fmt.Fprintf(dockerCli.Err(), "failed to remove %s: %v\n", link, err)
				} else {
					fmt.Fprintf(dockerCli.Out(), "app symlink removed %s\n", link)
				}
			}
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to remove some apps: %v", failed)
	}
	return nil
}
