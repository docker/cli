package container

import (
	"strings"
	"sync"

	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/container"
	"github.com/moby/sys/capability"
	"github.com/moby/sys/signal"
	"github.com/spf13/cobra"
)

// allCaps is the magic value for "all capabilities".
const allCaps = "ALL"

// allLinuxCapabilities is a list of all known Linux capabilities.
//
// TODO(thaJeztah): add descriptions, and enable descriptions for our completion scripts (cobra.CompletionOptions.DisableDescriptions is currently set to "true")
// TODO(thaJeztah): consider what casing we want to use for completion (see below);
//
// We need to consider what format is most convenient; currently we use the
// canonical name (uppercase and "CAP_" prefix), however, tab-completion is
// case-sensitive by default, so requires the user to type uppercase letters
// to filter the list of options.
//
// Bash completion provides a `completion-ignore-case on` option to make completion
// case-insensitive (https://askubuntu.com/a/87066), but it looks to be a global
// option; the current cobra.CompletionOptions also don't provide this as an option
// to be used in the generated completion-script.
//
// Fish completion has `smartcase` (by default?) which matches any case if
// all of the input is lowercase.
//
// Zsh does not appear have a dedicated option, but allows setting matching-rules
// (see https://superuser.com/a/1092328).
var allLinuxCapabilities = sync.OnceValue(func() []string {
	caps := capability.ListKnown()
	out := make([]string, 0, len(caps)+1)
	out = append(out, allCaps)
	for _, c := range caps {
		out = append(out, "CAP_"+strings.ToUpper(c.String()))
	}
	return out
})

// restartPolicies is a list of all valid restart-policies..
//
// TODO(thaJeztah): add descriptions, and enable descriptions for our completion scripts (cobra.CompletionOptions.DisableDescriptions is currently set to "true")
var restartPolicies = []string{
	string(container.RestartPolicyDisabled),
	string(container.RestartPolicyAlways),
	string(container.RestartPolicyOnFailure),
	string(container.RestartPolicyUnlessStopped),
}

// addCompletions adds the completions that `run` and `create` have in common.
func addCompletions(cmd *cobra.Command, dockerCLI completion.APIClientProvider) {
	_ = cmd.RegisterFlagCompletionFunc("attach", completion.FromList("stderr", "stdin", "stdout"))
	_ = cmd.RegisterFlagCompletionFunc("cap-add", completeLinuxCapabilityNames)
	_ = cmd.RegisterFlagCompletionFunc("cap-drop", completeLinuxCapabilityNames)
	_ = cmd.RegisterFlagCompletionFunc("env", completion.EnvVarNames)
	_ = cmd.RegisterFlagCompletionFunc("env-file", completion.FileNames)
	_ = cmd.RegisterFlagCompletionFunc("ipc", completeIpc(dockerCLI))
	_ = cmd.RegisterFlagCompletionFunc("link", completeLink(dockerCLI))
	_ = cmd.RegisterFlagCompletionFunc("network", completion.NetworkNames(dockerCLI))
	_ = cmd.RegisterFlagCompletionFunc("pid", completePid(dockerCLI))
	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)
	_ = cmd.RegisterFlagCompletionFunc("pull", completion.FromList(PullImageAlways, PullImageMissing, PullImageNever))
	_ = cmd.RegisterFlagCompletionFunc("restart", completeRestartPolicies)
	_ = cmd.RegisterFlagCompletionFunc("stop-signal", completeSignals)
	_ = cmd.RegisterFlagCompletionFunc("storage-opt", completeStorageOpt)
	_ = cmd.RegisterFlagCompletionFunc("volumes-from", completion.ContainerNames(dockerCLI, true))
}

// completeIpc implements shell completion for the `--ipc` option of `run` and `create`.
// The completion is partly composite.
func completeIpc(dockerCLI completion.APIClientProvider) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(toComplete) > 0 && strings.HasPrefix("container", toComplete) { //nolint:gocritic // not swapped, matches partly typed "container"
			return []string{"container:"}, cobra.ShellCompDirectiveNoSpace
		}
		if strings.HasPrefix(toComplete, "container:") {
			names, _ := completion.ContainerNames(dockerCLI, true)(cmd, args, toComplete)
			return prefixWith("container:", names), cobra.ShellCompDirectiveNoFileComp
		}
		return []string{
			string(container.IPCModeContainer + ":"),
			string(container.IPCModeHost),
			string(container.IPCModeNone),
			string(container.IPCModePrivate),
			string(container.IPCModeShareable),
		}, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeLink implements shell completion for the `--link` option  of `run` and `create`.
func completeLink(dockerCLI completion.APIClientProvider) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return postfixWith(":", containerNames(dockerCLI, cmd, args, toComplete)), cobra.ShellCompDirectiveNoSpace
	}
}

// completePid implements shell completion for the `--pid` option  of `run` and `create`.
func completePid(dockerCLI completion.APIClientProvider) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(toComplete) > 0 && strings.HasPrefix("container", toComplete) { //nolint:gocritic // not swapped, matches partly typed "container"
			return []string{"container:"}, cobra.ShellCompDirectiveNoSpace
		}
		if strings.HasPrefix(toComplete, "container:") {
			names, _ := completion.ContainerNames(dockerCLI, true)(cmd, args, toComplete)
			return prefixWith("container:", names), cobra.ShellCompDirectiveNoFileComp
		}
		return []string{"container:", "host"}, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeStorageOpt implements shell completion for the `--storage-opt` option  of `run` and `create`.
func completeStorageOpt(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return []string{"size="}, cobra.ShellCompDirectiveNoSpace
}

// containerNames contacts the API to get names and optionally IDs of containers.
// In case of an error, an empty list is returned.
func containerNames(dockerCLI completion.APIClientProvider, cmd *cobra.Command, args []string, toComplete string) []string {
	names, _ := completion.ContainerNames(dockerCLI, true)(cmd, args, toComplete)
	if names == nil {
		return []string{}
	}
	return names
}

// prefixWith prefixes every element in the slice with the given prefix.
func prefixWith(prefix string, values []string) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = prefix + v
	}
	return result
}

// postfixWith appends postfix to every element in the slice.
func postfixWith(postfix string, values []string) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v + postfix
	}
	return result
}

func completeLinuxCapabilityNames(cmd *cobra.Command, args []string, toComplete string) (names []string, _ cobra.ShellCompDirective) {
	return completion.FromList(allLinuxCapabilities()...)(cmd, args, toComplete)
}

func completeRestartPolicies(cmd *cobra.Command, args []string, toComplete string) (names []string, _ cobra.ShellCompDirective) {
	return completion.FromList(restartPolicies...)(cmd, args, toComplete)
}

func completeSignals(cmd *cobra.Command, args []string, toComplete string) (names []string, _ cobra.ShellCompDirective) {
	// TODO(thaJeztah): do we want to provide the full list here, or a subset?
	signalNames := make([]string, 0, len(signal.SignalMap))
	for k := range signal.SignalMap {
		signalNames = append(signalNames, k)
	}
	return completion.FromList(signalNames...)(cmd, args, toComplete)
}
