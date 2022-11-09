package main

import (
	"os"
	"strings"

	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	keyBuilderAlias = "builder"
)

var allowedAliases = map[string]struct{}{
	keyBuilderAlias: {},
}

func processAliases(dockerCli command.Cli, cmd *cobra.Command, args, osArgs []string) ([]string, []string, []string, error) {
	var err error
	var envs []string
	aliasMap := dockerCli.ConfigFile().Aliases
	aliases := make([][2][]string, 0, len(aliasMap))

	for k, v := range aliasMap {
		if _, ok := allowedAliases[k]; !ok {
			return args, osArgs, envs, errors.Errorf("not allowed to alias %q (allowed: %#v)", k, allowedAliases)
		}
		if c, _, err := cmd.Find(strings.Split(v, " ")); err == nil {
			if !pluginmanager.IsPluginCommand(c) {
				return args, osArgs, envs, errors.Errorf("not allowed to alias with builtin %q as target", v)
			}
		}
		aliases = append(aliases, [2][]string{{k}, {v}})
	}

	args, osArgs, envs, err = processBuilder(dockerCli, cmd, args, os.Args)
	if err != nil {
		return args, os.Args, envs, err
	}

	for _, al := range aliases {
		var didChange bool
		args, didChange = command.StringSliceReplaceAt(args, al[0], al[1], 0)
		if didChange {
			osArgs, _ = command.StringSliceReplaceAt(osArgs, al[0], al[1], -1)
			break
		}
	}

	return args, osArgs, envs, nil
}
