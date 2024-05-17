package manager

import (
	"encoding/json"
	"strings"

	"github.com/docker/cli/cli-plugins/hooks"
	"github.com/docker/cli/cli/command"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// HookPluginData is the type representing the information
// that plugins declaring support for hooks get passed when
// being invoked following a CLI command execution.
type HookPluginData struct {
	// RootCmd is a string representing the matching hook configuration
	// which is currently being invoked. If a hook for `docker context` is
	// configured and the user executes `docker context ls`, the plugin will
	// be invoked with `context`.
	RootCmd      string
	Flags        map[string]string
	CommandError string
}

// RunCLICommandHooks is the entrypoint into the hooks execution flow after
// a main CLI command was executed. It calls the hook subcommand for all
// present CLI plugins that declare support for hooks in their metadata and
// parses/prints their responses.
func RunCLICommandHooks(dockerCli command.Cli, rootCmd, subCommand *cobra.Command, cmdErrorMessage string) {
	commandName := strings.TrimPrefix(subCommand.CommandPath(), rootCmd.Name()+" ")
	flags := getCommandFlags(subCommand)

	runHooks(dockerCli, rootCmd, subCommand, commandName, flags, cmdErrorMessage)
}

// RunPluginHooks is the entrypoint for the hooks execution flow
// after a plugin command was just executed by the CLI.
func RunPluginHooks(dockerCli command.Cli, rootCmd, subCommand *cobra.Command, args []string) {
	commandName := strings.Join(args, " ")
	flags := getNaiveFlags(args)

	runHooks(dockerCli, rootCmd, subCommand, commandName, flags, "")
}

func runHooks(dockerCli command.Cli, rootCmd, subCommand *cobra.Command, invokedCommand string, flags map[string]string, cmdErrorMessage string) {
	nextSteps := invokeAndCollectHooks(dockerCli, rootCmd, subCommand, invokedCommand, flags, cmdErrorMessage)

	hooks.PrintNextSteps(dockerCli.Err(), nextSteps)
}

func invokeAndCollectHooks(dockerCli command.Cli, rootCmd, subCmd *cobra.Command, subCmdStr string, flags map[string]string, cmdErrorMessage string) []string {
	pluginsCfg := dockerCli.ConfigFile().Plugins
	if pluginsCfg == nil {
		return nil
	}

	nextSteps := make([]string, 0, len(pluginsCfg))
	for pluginName, cfg := range pluginsCfg {
		match, ok := pluginMatch(cfg, subCmdStr)
		if !ok {
			continue
		}

		p, err := GetPlugin(pluginName, dockerCli, rootCmd)
		if err != nil {
			continue
		}

		hookReturn, err := p.RunHook(HookPluginData{
			RootCmd:      match,
			Flags:        flags,
			CommandError: cmdErrorMessage,
		})
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

		var appended bool
		nextSteps, appended = appendNextSteps(nextSteps, processedHook)
		if !appended {
			logrus.Debugf("Plugin %s responded with an empty hook message %q. Ignoring.", pluginName, string(hookReturn))
		}
	}
	return nextSteps
}

// appendNextSteps appends the processed hook output to the nextSteps slice.
// If the processed hook output is empty, it is not appended.
// Empty lines are not stripped if there's at least one non-empty line.
func appendNextSteps(nextSteps []string, processed []string) ([]string, bool) {
	empty := true
	for _, l := range processed {
		if strings.TrimSpace(l) != "" {
			empty = false
			break
		}
	}

	if empty {
		return nextSteps, false
	}

	return append(nextSteps, processed...), true
}

// pluginMatch takes a plugin configuration and a string representing the
// command being executed (such as 'image ls' – the root 'docker' is omitted)
// and, if the configuration includes a hook for the invoked command, returns
// the configured hook string.
func pluginMatch(pluginCfg map[string]string, subCmd string) (string, bool) {
	configuredPluginHooks, ok := pluginCfg["hooks"]
	if !ok || configuredPluginHooks == "" {
		return "", false
	}

	commands := strings.Split(configuredPluginHooks, ",")
	for _, hookCmd := range commands {
		if hookMatch(hookCmd, subCmd) {
			return hookCmd, true
		}
	}

	return "", false
}

func hookMatch(hookCmd, subCmd string) bool {
	hookCmdTokens := strings.Split(hookCmd, " ")
	subCmdTokens := strings.Split(subCmd, " ")

	if len(hookCmdTokens) > len(subCmdTokens) {
		return false
	}

	for i, v := range hookCmdTokens {
		if v != subCmdTokens[i] {
			return false
		}
	}

	return true
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
