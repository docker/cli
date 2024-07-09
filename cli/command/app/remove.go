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
	targets, err := findSymlinks(binPath)
	if err != nil {
		return err
	}

	var failed []string

	for _, app := range apps {
		options.setArgs([]string{app})
		appPath, err := options.appPath()
		if err != nil {
			failed = append(failed, app)
			continue
		}

		// optionally run uninstall if provided
		envs, _ := options.makeEnvs()
		runUninstaller(dockerCli, appPath, envs)

		// remove all files under the app path
		if err := os.RemoveAll(appPath); err != nil {
			failed = append(failed, app)
			continue
		}
		removeEmptyPath(options.pkgPath(), appPath)
		fmt.Fprintf(dockerCli.Out(), "app package removed %s\n", appPath)

		cleanupSymlink(dockerCli, appPath, targets)
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to remove some apps: %v", failed)
	}
	return nil
}

// find all symlinks in binPath for removal
func findSymlinks(binPath string) (map[string]string, error) {
	targets := make(map[string]string)
	readlink := func(link string) (string, error) {
		target, err := os.Readlink(link)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(link), target)
		}
		abs, err := filepath.Abs(target)
		if err != nil {
			return "", err
		}
		return abs, nil
	}
	if links, err := findLinks(binPath); err == nil {
		for _, link := range links {
			if target, err := readlink(link); err == nil {
				targets[target] = link
			} else {
				return nil, err
			}
		}
	}
	return targets, nil
}

// runUninstaller optionally runs uninstall if provided
func runUninstaller(dockerCli command.Cli, appPath string, envs map[string]string) {
	uninstaller := filepath.Join(appPath, uninstallerName)
	if _, err := os.Stat(uninstaller); err == nil {
		err := spawn(uninstaller, nil, envs, false)
		if err != nil {
			fmt.Fprintf(dockerCli.Err(), "%s failed to run: %v\n", uninstaller, err)
		}
	}
}

// cleanupSymlink removes symlinks of the app if any
func cleanupSymlink(dockerCli command.Cli, appPath string, targets map[string]string) {
	owns := func(app, target string) bool {
		return strings.Contains(target, app)
	}
	for target, link := range targets {
		if owns(appPath, target) {
			if err := os.Remove(link); err != nil {
				fmt.Fprintf(dockerCli.Err(), "failed to remove %s: %v\n", link, err)
			} else {
				fmt.Fprintf(dockerCli.Out(), "app symlink removed %s\n", link)
			}
		}
	}
}
