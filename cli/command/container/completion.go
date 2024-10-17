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
