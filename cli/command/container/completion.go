package container

import (
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/container"
	"github.com/moby/sys/signal"
	"github.com/spf13/cobra"
)

// allLinuxCapabilities is a list of all known Linux capabilities.
//
// This list was based on the containerd pkg/cap package;
// https://github.com/containerd/containerd/blob/v1.7.19/pkg/cap/cap_linux.go#L133-L181
//
// TODO(thaJeztah): add descriptions, and enable descriptions for our completion scripts (cobra.CompletionOptions.DisableDescriptions is currently set to "true")
var allLinuxCapabilities = []string{
	"ALL", // magic value for "all capabilities"

	// caps35 is the caps of kernel 3.5 (37 entries)
	"CAP_CHOWN",            // 2.2
	"CAP_DAC_OVERRIDE",     // 2.2
	"CAP_DAC_READ_SEARCH",  // 2.2
	"CAP_FOWNER",           // 2.2
	"CAP_FSETID",           // 2.2
	"CAP_KILL",             // 2.2
	"CAP_SETGID",           // 2.2
	"CAP_SETUID",           // 2.2
	"CAP_SETPCAP",          // 2.2
	"CAP_LINUX_IMMUTABLE",  // 2.2
	"CAP_NET_BIND_SERVICE", // 2.2
	"CAP_NET_BROADCAST",    // 2.2
	"CAP_NET_ADMIN",        // 2.2
	"CAP_NET_RAW",          // 2.2
	"CAP_IPC_LOCK",         // 2.2
	"CAP_IPC_OWNER",        // 2.2
	"CAP_SYS_MODULE",       // 2.2
	"CAP_SYS_RAWIO",        // 2.2
	"CAP_SYS_CHROOT",       // 2.2
	"CAP_SYS_PTRACE",       // 2.2
	"CAP_SYS_PACCT",        // 2.2
	"CAP_SYS_ADMIN",        // 2.2
	"CAP_SYS_BOOT",         // 2.2
	"CAP_SYS_NICE",         // 2.2
	"CAP_SYS_RESOURCE",     // 2.2
	"CAP_SYS_TIME",         // 2.2
	"CAP_SYS_TTY_CONFIG",   // 2.2
	"CAP_MKNOD",            // 2.4
	"CAP_LEASE",            // 2.4
	"CAP_AUDIT_WRITE",      // 2.6.11
	"CAP_AUDIT_CONTROL",    // 2.6.11
	"CAP_SETFCAP",          // 2.6.24
	"CAP_MAC_OVERRIDE",     // 2.6.25
	"CAP_MAC_ADMIN",        // 2.6.25
	"CAP_SYSLOG",           // 2.6.37
	"CAP_WAKE_ALARM",       // 3.0
	"CAP_BLOCK_SUSPEND",    // 3.5

	// caps316 is the caps of kernel 3.16 (38 entries)
	"CAP_AUDIT_READ",

	// caps58 is the caps of kernel 5.8 (40 entries)
	"CAP_PERFMON",
	"CAP_BPF",

	// caps59 is the caps of kernel 5.9 (41 entries)
	"CAP_CHECKPOINT_RESTORE",
}

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
	return completion.FromList(allLinuxCapabilities...)(cmd, args, toComplete)
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
