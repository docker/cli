package manager

import (
	"encoding/json"
	"strings"

	"github.com/docker/cli/cli-plugins/hooks"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// HookPluginData is the type representing the information
// that plugins declaring support for hooks get passed when
// being invoked following a CLI command execution.
type HookPluginData struct {
	RootCmd string
	Flags   map[string]string
}

// RunPluginHooks calls the hook subcommand for all present
// CLI plugins that declare support for hooks in their metadata
// and parses/prints their responses.
func RunPluginHooks(dockerCli command.Cli, rootCmd, subCommand *cobra.Command, plugin string, args []string) error {
	subCmdName := subCommand.Name()
	if plugin != "" {
		subCmdName = plugin
	}
	var flags map[string]string
	if plugin == "" {
		flags = getCommandFlags(subCommand)
	} else {
		flags = getNaiveFlags(args)
	}
	nextSteps := invokeAndCollectHooks(dockerCli, rootCmd, subCommand, subCmdName, flags)

	hooks.PrintNextSteps(dockerCli.Err(), nextSteps)
	return nil
}

func invokeAndCollectHooks(dockerCli command.Cli, rootCmd, subCmd *cobra.Command, hookCmdName string, flags map[string]string) []string {
	pluginsCfg := dockerCli.ConfigFile().Plugins
	if pluginsCfg == nil {
		return nil
	}

	nextSteps := make([]string, 0, len(pluginsCfg))
	for pluginName, cfg := range pluginsCfg {
		if !registersHook(cfg, hookCmdName) {
			continue
		}

		p, err := GetPlugin(pluginName, dockerCli, rootCmd)
		if err != nil {
			continue
		}

		hookReturn, err := p.RunHook(hookCmdName, flags)
		if err != nil {
			// skip misbehaving plugins, but don't halt execution
			continue
		}

		var hookMessageData hooks.HookMessage
		err = json.Unmarshal(hookReturn, &hookMessageData)
		if err != nil {
			continue
		}

		// currently the only hook type
		if hookMessageData.Type != hooks.NextSteps {
			continue
		}

		processedHook, err := hooks.ParseTemplate(hookMessageData.Template, subCmd)
		if err != nil {
			continue
		}
		nextSteps = append(nextSteps, processedHook)
	}
	return nextSteps
}

func registersHook(pluginCfg map[string]string, subCmdName string) bool {
	hookCmdStr, ok := pluginCfg["hooks"]
	if !ok {
		return false
	}
	commands := strings.Split(hookCmdStr, ",")
	for _, hookCmd := range commands {
		if hookCmd == subCmdName {
			return true
		}
	}
	return false
}

func getCommandFlags(cmd *cobra.Command) map[string]string {
	flags := make(map[string]string)
	cmd.Flags().Visit(func(f *pflag.Flag) {
		var fValue string
		if f.Value.Type() == "bool" {
			fValue = f.Value.String()
		}
		flags[f.Name] = fValue
	})
	return flags
}

// getNaiveFlags string-matches argv and parses them into a map.
// This is used when calling hooks after a plugin command, since
// in this case we can't rely on the cobra command tree to parse
// flags in this case. In this case, no values are ever passed,
// since we don't have enough information to process them.
func getNaiveFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			flags[arg[2:]] = ""
			continue
		}
		if strings.HasPrefix(arg, "-") {
			flags[arg[1:]] = ""
		}
	}
	return flags
}
