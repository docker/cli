package system

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/debug"
	"github.com/docker/cli/templates"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type infoOptions struct {
	format string
}

// NewInfoCommand creates a new cobra.Command for `docker info`
func NewInfoCommand(dockerCli command.Cli) *cobra.Command {
	var opts infoOptions

	cmd := &cobra.Command{
		Use:   "info [OPTIONS]",
		Short: "Display system-wide information",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")

	return cmd
}

func runInfo(dockerCli command.Cli, opts *infoOptions) error {
	ctx := context.Background()
	info, err := dockerCli.Client().Info(ctx)
	if err != nil {
		return err
	}
	if opts.format == "" {
		return prettyPrintInfo(dockerCli, info)
	}
	return formatInfo(dockerCli, info, opts.format)
}

// nolint: gocyclo
func prettyPrintInfo(dockerCli command.Cli, info types.Info) error {
	fmt.Fprintln(dockerCli.Out(), "Containers:", info.Containers)
	fmt.Fprintln(dockerCli.Out(), " Running:", info.ContainersRunning)
	fmt.Fprintln(dockerCli.Out(), " Paused:", info.ContainersPaused)
	fmt.Fprintln(dockerCli.Out(), " Stopped:", info.ContainersStopped)
	fmt.Fprintln(dockerCli.Out(), "Images:", info.Images)
	fprintlnNonEmpty(dockerCli.Out(), "Server Version:", info.ServerVersion)
	fprintlnNonEmpty(dockerCli.Out(), "Storage Driver:", info.Driver)
	if info.DriverStatus != nil {
		for _, pair := range info.DriverStatus {
			fmt.Fprintf(dockerCli.Out(), " %s: %s\n", pair[0], pair[1])
		}
	}
	if info.SystemStatus != nil {
		for _, pair := range info.SystemStatus {
			fmt.Fprintf(dockerCli.Out(), "%s: %s\n", pair[0], pair[1])
		}
	}
	fprintlnNonEmpty(dockerCli.Out(), "Logging Driver:", info.LoggingDriver)
	fprintlnNonEmpty(dockerCli.Out(), "Cgroup Driver:", info.CgroupDriver)

	fmt.Fprintln(dockerCli.Out(), "Plugins:")
	fmt.Fprintln(dockerCli.Out(), " Volume:", strings.Join(info.Plugins.Volume, " "))
	fmt.Fprintln(dockerCli.Out(), " Network:", strings.Join(info.Plugins.Network, " "))

	if len(info.Plugins.Authorization) != 0 {
		fmt.Fprintln(dockerCli.Out(), " Authorization:", strings.Join(info.Plugins.Authorization, " "))
	}

	fmt.Fprintln(dockerCli.Out(), " Log:", strings.Join(info.Plugins.Log, " "))

	fmt.Fprintln(dockerCli.Out(), "Swarm:", info.Swarm.LocalNodeState)
	printSwarmInfo(dockerCli, info)

	if len(info.Runtimes) > 0 {
		fmt.Fprint(dockerCli.Out(), "Runtimes:")
		for name := range info.Runtimes {
			fmt.Fprintf(dockerCli.Out(), " %s", name)
		}
		fmt.Fprint(dockerCli.Out(), "\n")
		fmt.Fprintln(dockerCli.Out(), "Default Runtime:", info.DefaultRuntime)
	}

	if info.OSType == "linux" {
		fmt.Fprintln(dockerCli.Out(), "Init Binary:", info.InitBinary)

		for _, ci := range []struct {
			Name   string
			Commit types.Commit
		}{
			{"containerd", info.ContainerdCommit},
			{"runc", info.RuncCommit},
			{"init", info.InitCommit},
		} {
			fmt.Fprintf(dockerCli.Out(), "%s version: %s", ci.Name, ci.Commit.ID)
			if ci.Commit.ID != ci.Commit.Expected {
				fmt.Fprintf(dockerCli.Out(), " (expected: %s)", ci.Commit.Expected)
			}
			fmt.Fprint(dockerCli.Out(), "\n")
		}
		if len(info.SecurityOptions) != 0 {
			kvs, err := types.DecodeSecurityOptions(info.SecurityOptions)
			if err != nil {
				return err
			}
			fmt.Fprintln(dockerCli.Out(), "Security Options:")
			for _, so := range kvs {
				fmt.Fprintln(dockerCli.Out(), " "+so.Name)
				for _, o := range so.Options {
					switch o.Key {
					case "profile":
						if o.Value != "default" {
							fmt.Fprintln(dockerCli.Err(), "  WARNING: You're not using the default seccomp profile")
						}
						fmt.Fprintln(dockerCli.Out(), "  Profile:", o.Value)
					}
				}
			}
		}
	}

	// Isolation only has meaning on a Windows daemon.
	if info.OSType == "windows" {
		fmt.Fprintln(dockerCli.Out(), "Default Isolation:", info.Isolation)
	}

	fprintlnNonEmpty(dockerCli.Out(), "Kernel Version:", info.KernelVersion)
	fprintlnNonEmpty(dockerCli.Out(), "Operating System:", info.OperatingSystem)
	fprintlnNonEmpty(dockerCli.Out(), "OSType:", info.OSType)
	fprintlnNonEmpty(dockerCli.Out(), "Architecture:", info.Architecture)
	fmt.Fprintln(dockerCli.Out(), "CPUs:", info.NCPU)
	fmt.Fprintln(dockerCli.Out(), "Total Memory:", units.BytesSize(float64(info.MemTotal)))
	fprintlnNonEmpty(dockerCli.Out(), "Name:", info.Name)
	fprintlnNonEmpty(dockerCli.Out(), "ID:", info.ID)
	fmt.Fprintln(dockerCli.Out(), "Docker Root Dir:", info.DockerRootDir)
	fmt.Fprintln(dockerCli.Out(), "Debug Mode (client):", debug.IsEnabled())
	fmt.Fprintln(dockerCli.Out(), "Debug Mode (server):", info.Debug)

	if info.Debug {
		fmt.Fprintln(dockerCli.Out(), " File Descriptors:", info.NFd)
		fmt.Fprintln(dockerCli.Out(), " Goroutines:", info.NGoroutines)
		fmt.Fprintln(dockerCli.Out(), " System Time:", info.SystemTime)
		fmt.Fprintln(dockerCli.Out(), " EventsListeners:", info.NEventsListener)
	}

	fprintlnNonEmpty(dockerCli.Out(), "HTTP Proxy:", info.HTTPProxy)
	fprintlnNonEmpty(dockerCli.Out(), "HTTPS Proxy:", info.HTTPSProxy)
	fprintlnNonEmpty(dockerCli.Out(), "No Proxy:", info.NoProxy)

	if info.IndexServerAddress != "" {
		u := dockerCli.ConfigFile().AuthConfigs[info.IndexServerAddress].Username
		if len(u) > 0 {
			fmt.Fprintln(dockerCli.Out(), "Username:", u)
		}
		fmt.Fprintln(dockerCli.Out(), "Registry:", info.IndexServerAddress)
	}

	if info.Labels != nil {
		fmt.Fprintln(dockerCli.Out(), "Labels:")
		for _, lbl := range info.Labels {
			fmt.Fprintln(dockerCli.Out(), " "+lbl)
		}
	}

	fmt.Fprintln(dockerCli.Out(), "Experimental:", info.ExperimentalBuild)
	fprintlnNonEmpty(dockerCli.Out(), "Cluster Store:", info.ClusterStore)
	fprintlnNonEmpty(dockerCli.Out(), "Cluster Advertise:", info.ClusterAdvertise)

	if info.RegistryConfig != nil && (len(info.RegistryConfig.InsecureRegistryCIDRs) > 0 || len(info.RegistryConfig.IndexConfigs) > 0) {
		fmt.Fprintln(dockerCli.Out(), "Insecure Registries:")
		for _, registry := range info.RegistryConfig.IndexConfigs {
			if !registry.Secure {
				fmt.Fprintln(dockerCli.Out(), " "+registry.Name)
			}
		}

		for _, registry := range info.RegistryConfig.InsecureRegistryCIDRs {
			mask, _ := registry.Mask.Size()
			fmt.Fprintf(dockerCli.Out(), " %s/%d\n", registry.IP.String(), mask)
		}
	}

	if info.RegistryConfig != nil && len(info.RegistryConfig.Mirrors) > 0 {
		fmt.Fprintln(dockerCli.Out(), "Registry Mirrors:")
		for _, mirror := range info.RegistryConfig.Mirrors {
			fmt.Fprintln(dockerCli.Out(), " "+mirror)
		}
	}

	fmt.Fprintln(dockerCli.Out(), "Live Restore Enabled:", info.LiveRestoreEnabled)
	fmt.Fprint(dockerCli.Out(), "\n")

	// Only output these warnings if the server does not support these features
	if info.OSType != "windows" {
		printStorageDriverWarnings(dockerCli, info)

		if !info.MemoryLimit {
			fmt.Fprintln(dockerCli.Err(), "WARNING: No memory limit support")
		}
		if !info.SwapLimit {
			fmt.Fprintln(dockerCli.Err(), "WARNING: No swap limit support")
		}
		if !info.KernelMemory {
			fmt.Fprintln(dockerCli.Err(), "WARNING: No kernel memory limit support")
		}
		if !info.OomKillDisable {
			fmt.Fprintln(dockerCli.Err(), "WARNING: No oom kill disable support")
		}
		if !info.CPUCfsQuota {
			fmt.Fprintln(dockerCli.Err(), "WARNING: No cpu cfs quota support")
		}
		if !info.CPUCfsPeriod {
			fmt.Fprintln(dockerCli.Err(), "WARNING: No cpu cfs period support")
		}
		if !info.CPUShares {
			fmt.Fprintln(dockerCli.Err(), "WARNING: No cpu shares support")
		}
		if !info.CPUSet {
			fmt.Fprintln(dockerCli.Err(), "WARNING: No cpuset support")
		}
		if !info.IPv4Forwarding {
			fmt.Fprintln(dockerCli.Err(), "WARNING: IPv4 forwarding is disabled")
		}
		if !info.BridgeNfIptables {
			fmt.Fprintln(dockerCli.Err(), "WARNING: bridge-nf-call-iptables is disabled")
		}
		if !info.BridgeNfIP6tables {
			fmt.Fprintln(dockerCli.Err(), "WARNING: bridge-nf-call-ip6tables is disabled")
		}
	}

	return nil
}

func printSwarmInfo(dockerCli command.Cli, info types.Info) {
	if info.Swarm.LocalNodeState == swarm.LocalNodeStateInactive || info.Swarm.LocalNodeState == swarm.LocalNodeStateLocked {
		return
	}
	fmt.Fprintln(dockerCli.Out(), " NodeID:", info.Swarm.NodeID)
	if info.Swarm.Error != "" {
		fmt.Fprintln(dockerCli.Out(), " Error:", info.Swarm.Error)
	}
	fmt.Fprintln(dockerCli.Out(), " Is Manager:", info.Swarm.ControlAvailable)
	if info.Swarm.Cluster != nil && info.Swarm.ControlAvailable && info.Swarm.Error == "" && info.Swarm.LocalNodeState != swarm.LocalNodeStateError {
		fmt.Fprintln(dockerCli.Out(), " ClusterID:", info.Swarm.Cluster.ID)
		fmt.Fprintln(dockerCli.Out(), " Managers:", info.Swarm.Managers)
		fmt.Fprintln(dockerCli.Out(), " Nodes:", info.Swarm.Nodes)
		fmt.Fprintln(dockerCli.Out(), " Orchestration:")
		taskHistoryRetentionLimit := int64(0)
		if info.Swarm.Cluster.Spec.Orchestration.TaskHistoryRetentionLimit != nil {
			taskHistoryRetentionLimit = *info.Swarm.Cluster.Spec.Orchestration.TaskHistoryRetentionLimit
		}
		fmt.Fprintln(dockerCli.Out(), "  Task History Retention Limit:", taskHistoryRetentionLimit)
		fmt.Fprintln(dockerCli.Out(), " Raft:")
		fmt.Fprintln(dockerCli.Out(), "  Snapshot Interval:", info.Swarm.Cluster.Spec.Raft.SnapshotInterval)
		if info.Swarm.Cluster.Spec.Raft.KeepOldSnapshots != nil {
			fmt.Fprintf(dockerCli.Out(), "  Number of Old Snapshots to Retain: %d\n", *info.Swarm.Cluster.Spec.Raft.KeepOldSnapshots)
		}
		fmt.Fprintln(dockerCli.Out(), "  Heartbeat Tick:", info.Swarm.Cluster.Spec.Raft.HeartbeatTick)
		fmt.Fprintln(dockerCli.Out(), "  Election Tick:", info.Swarm.Cluster.Spec.Raft.ElectionTick)
		fmt.Fprintln(dockerCli.Out(), " Dispatcher:")
		fmt.Fprintln(dockerCli.Out(), "  Heartbeat Period:", units.HumanDuration(info.Swarm.Cluster.Spec.Dispatcher.HeartbeatPeriod))
		fmt.Fprintln(dockerCli.Out(), " CA Configuration:")
		fmt.Fprintln(dockerCli.Out(), "  Expiry Duration:", units.HumanDuration(info.Swarm.Cluster.Spec.CAConfig.NodeCertExpiry))
		fmt.Fprintln(dockerCli.Out(), "  Force Rotate:", info.Swarm.Cluster.Spec.CAConfig.ForceRotate)
		if caCert := strings.TrimSpace(info.Swarm.Cluster.Spec.CAConfig.SigningCACert); caCert != "" {
			fmt.Fprintf(dockerCli.Out(), "  Signing CA Certificate: \n%s\n\n", caCert)
		}
		if len(info.Swarm.Cluster.Spec.CAConfig.ExternalCAs) > 0 {
			fmt.Fprintln(dockerCli.Out(), "  External CAs:")
			for _, entry := range info.Swarm.Cluster.Spec.CAConfig.ExternalCAs {
				fmt.Fprintf(dockerCli.Out(), "    %s: %s\n", entry.Protocol, entry.URL)
			}
		}
		fmt.Fprintln(dockerCli.Out(), " Autolock Managers:", info.Swarm.Cluster.Spec.EncryptionConfig.AutoLockManagers)
		fmt.Fprintln(dockerCli.Out(), " Root Rotation In Progress:", info.Swarm.Cluster.RootRotationInProgress)
	}
	fmt.Fprintln(dockerCli.Out(), " Node Address:", info.Swarm.NodeAddr)
	if len(info.Swarm.RemoteManagers) > 0 {
		managers := []string{}
		for _, entry := range info.Swarm.RemoteManagers {
			managers = append(managers, entry.Addr)
		}
		sort.Strings(managers)
		fmt.Fprintln(dockerCli.Out(), " Manager Addresses:")
		for _, entry := range managers {
			fmt.Fprintf(dockerCli.Out(), "  %s\n", entry)
		}
	}
}

func printStorageDriverWarnings(dockerCli command.Cli, info types.Info) {
	if info.DriverStatus == nil {
		return
	}

	for _, pair := range info.DriverStatus {
		if pair[0] == "Data loop file" {
			fmt.Fprintf(dockerCli.Err(), "WARNING: %s: usage of loopback devices is "+
				"strongly discouraged for production use.\n         "+
				"Use `--storage-opt dm.thinpooldev` to specify a custom block storage device.\n", info.Driver)
		}
		if pair[0] == "Supports d_type" && pair[1] == "false" {
			backingFs := getBackingFs(info)

			msg := fmt.Sprintf("WARNING: %s: the backing %s filesystem is formatted without d_type support, which leads to incorrect behavior.\n", info.Driver, backingFs)
			if backingFs == "xfs" {
				msg += "         Reformat the filesystem with ftype=1 to enable d_type support.\n"
			}
			msg += "         Running without d_type support will not be supported in future releases."
			fmt.Fprintln(dockerCli.Err(), msg)
		}
	}
}

func getBackingFs(info types.Info) string {
	if info.DriverStatus == nil {
		return ""
	}

	for _, pair := range info.DriverStatus {
		if pair[0] == "Backing Filesystem" {
			return pair[1]
		}
	}
	return ""
}

func formatInfo(dockerCli command.Cli, info types.Info, format string) error {
	tmpl, err := templates.Parse(format)
	if err != nil {
		return cli.StatusError{StatusCode: 64,
			Status: "Template parsing error: " + err.Error()}
	}
	err = tmpl.Execute(dockerCli.Out(), info)
	dockerCli.Out().Write([]byte{'\n'})
	return err
}

func fprintlnNonEmpty(w io.Writer, label, value string) {
	if value != "" {
		fmt.Fprintln(w, label, value)
	}
}
