package system

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/docker/cli/cli"
	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/debug"
	flagsHelper "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/templates"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/registry"
	"github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type infoOptions struct {
	format string
}

type clientInfo struct {
	Debug bool
	clientVersion
	Plugins  []pluginmanager.Plugin
	Warnings []string
}

type info struct {
	// This field should/could be ServerInfo but is anonymous to
	// preserve backwards compatibility in the JSON rendering
	// which has ServerInfo immediately within the top-level
	// object.
	*types.Info  `json:",omitempty"`
	ServerErrors []string `json:",omitempty"`
	UserName     string   `json:"-"`

	ClientInfo   *clientInfo `json:",omitempty"`
	ClientErrors []string    `json:",omitempty"`
}

func (i *info) clientPlatform() string {
	if i.ClientInfo != nil && i.ClientInfo.Platform != nil {
		return i.ClientInfo.Platform.Name
	}
	return ""
}

// NewInfoCommand creates a new cobra.Command for `docker info`
func NewInfoCommand(dockerCli command.Cli) *cobra.Command {
	var opts infoOptions

	cmd := &cobra.Command{
		Use:   "info [OPTIONS]",
		Short: "Display system-wide information",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInfo(cmd, dockerCli, &opts)
		},
		Annotations: map[string]string{
			"category-top": "12",
			"aliases":      "docker system info, docker info",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	cmd.Flags().StringVarP(&opts.format, "format", "f", "", flagsHelper.InspectFormatHelp)
	return cmd
}

func runInfo(cmd *cobra.Command, dockerCli command.Cli, opts *infoOptions) error {
	info := info{
		ClientInfo: &clientInfo{
			// Don't pass a dockerCLI to newClientVersion(), because we currently
			// don't include negotiated API version, and want to avoid making an
			// API connection when only printing the Client section.
			clientVersion: newClientVersion(dockerCli.CurrentContext(), nil),
			Debug:         debug.IsEnabled(),
		},
		Info: &types.Info{},
	}
	if plugins, err := pluginmanager.ListPlugins(dockerCli, cmd.Root()); err == nil {
		info.ClientInfo.Plugins = plugins
	} else {
		info.ClientErrors = append(info.ClientErrors, err.Error())
	}

	if needsServerInfo(opts.format, info) {
		ctx := context.Background()
		if dinfo, err := dockerCli.Client().Info(ctx); err == nil {
			info.Info = &dinfo
		} else {
			info.ServerErrors = append(info.ServerErrors, err.Error())
			if opts.format == "" {
				// reset the server info to prevent printing "empty" Server info
				// and warnings, but don't reset it if a custom format was specified
				// to prevent errors from Go's template parsing during format.
				info.Info = nil
			} else {
				// if a format is provided, print the error, as it may be hidden
				// otherwise if the template doesn't include the ServerErrors field.
				fmt.Fprintln(dockerCli.Err(), err)
			}
		}
	}

	if opts.format == "" {
		info.UserName = dockerCli.ConfigFile().AuthConfigs[registry.IndexServer].Username
		info.ClientInfo.APIVersion = dockerCli.CurrentVersion()
		return prettyPrintInfo(dockerCli, info)
	}
	return formatInfo(dockerCli, info, opts.format)
}

// placeHolders does a rudimentary match for possible placeholders in a
// template, matching a '.', followed by an letter (a-z/A-Z).
var placeHolders = regexp.MustCompile(`\.[a-zA-Z]`)

// needsServerInfo detects if the given template uses any server information.
// If only client-side information is used in the template, we can skip
// connecting to the daemon. This allows (e.g.) to only get cli-plugin
// information, without also making a (potentially expensive) API call.
func needsServerInfo(template string, info info) bool {
	if len(template) == 0 || placeHolders.FindString(template) == "" {
		// The template is empty, or does not contain formatting fields
		// (e.g. `table` or `raw` or `{{ json .}}`). Assume we need server-side
		// information to render it.
		return true
	}

	// A template is provided and has at least one field set.
	tmpl, err := templates.NewParse("", template)
	if err != nil {
		// ignore parsing errors here, and let regular code handle them
		return true
	}

	type sparseInfo struct {
		ClientInfo   *clientInfo `json:",omitempty"`
		ClientErrors []string    `json:",omitempty"`
	}

	// This constructs an "info" object that only has the client-side fields.
	err = tmpl.Execute(io.Discard, sparseInfo{
		ClientInfo:   info.ClientInfo,
		ClientErrors: info.ClientErrors,
	})
	// If executing the template failed, it means the template needs
	// server-side information as well. If it succeeded without server-side
	// information, we don't need to make API calls to collect that information.
	return err != nil
}

func prettyPrintInfo(dockerCli command.Cli, info info) error {
	// Only append the platform info if it's not empty, to prevent printing a trailing space.
	if p := info.clientPlatform(); p != "" {
		_, _ = fmt.Fprintln(dockerCli.Out(), "Client:", p)
	} else {
		_, _ = fmt.Fprintln(dockerCli.Out(), "Client:")
	}
	if info.ClientInfo != nil {
		prettyPrintClientInfo(dockerCli, *info.ClientInfo)
	}
	for _, err := range info.ClientErrors {
		fmt.Fprintln(dockerCli.Err(), "ERROR:", err)
	}

	fmt.Fprintln(dockerCli.Out())
	fmt.Fprintln(dockerCli.Out(), "Server:")
	if info.Info != nil {
		for _, err := range prettyPrintServerInfo(dockerCli, &info) {
			info.ServerErrors = append(info.ServerErrors, err.Error())
		}
	}
	for _, err := range info.ServerErrors {
		fmt.Fprintln(dockerCli.Err(), "ERROR:", err)
	}

	if len(info.ServerErrors) > 0 || len(info.ClientErrors) > 0 {
		return fmt.Errorf("errors pretty printing info")
	}
	return nil
}

func prettyPrintClientInfo(streams command.Streams, info clientInfo) {
	output := streams.Out()
	fprintlnNonEmpty(output, " Version:   ", info.Version)
	fmt.Fprintln(output, " Context:   ", info.Context)
	fmt.Fprintln(output, " Debug Mode:", info.Debug)

	if len(info.Plugins) > 0 {
		fmt.Fprintln(output, " Plugins:")
		for _, p := range info.Plugins {
			if p.Err == nil {
				fmt.Fprintf(output, "  %s: %s (%s)\n", p.Name, p.ShortDescription, p.Vendor)
				fprintlnNonEmpty(output, "    Version: ", p.Version)
				fprintlnNonEmpty(output, "    Path:    ", p.Path)
			} else {
				info.Warnings = append(info.Warnings, fmt.Sprintf("WARNING: Plugin %q is not valid: %s", p.Path, p.Err))
			}
		}
	}

	if len(info.Warnings) > 0 {
		fmt.Fprintln(streams.Err(), strings.Join(info.Warnings, "\n"))
	}
}

//nolint:gocyclo
func prettyPrintServerInfo(streams command.Streams, info *info) []error {
	var errs []error
	output := streams.Out()

	fmt.Fprintln(output, " Containers:", info.Containers)
	fmt.Fprintln(output, "  Running:", info.ContainersRunning)
	fmt.Fprintln(output, "  Paused:", info.ContainersPaused)
	fmt.Fprintln(output, "  Stopped:", info.ContainersStopped)
	fmt.Fprintln(output, " Images:", info.Images)
	fprintlnNonEmpty(output, " Server Version:", info.ServerVersion)
	fprintlnNonEmpty(output, " Storage Driver:", info.Driver)
	if info.DriverStatus != nil {
		for _, pair := range info.DriverStatus {
			fmt.Fprintf(output, "  %s: %s\n", pair[0], pair[1])
		}
	}
	if info.SystemStatus != nil {
		for _, pair := range info.SystemStatus {
			fmt.Fprintf(output, " %s: %s\n", pair[0], pair[1])
		}
	}
	fprintlnNonEmpty(output, " Logging Driver:", info.LoggingDriver)
	fprintlnNonEmpty(output, " Cgroup Driver:", info.CgroupDriver)
	fprintlnNonEmpty(output, " Cgroup Version:", info.CgroupVersion)

	fmt.Fprintln(output, " Plugins:")
	fmt.Fprintln(output, "  Volume:", strings.Join(info.Plugins.Volume, " "))
	fmt.Fprintln(output, "  Network:", strings.Join(info.Plugins.Network, " "))

	if len(info.Plugins.Authorization) != 0 {
		fmt.Fprintln(output, "  Authorization:", strings.Join(info.Plugins.Authorization, " "))
	}

	fmt.Fprintln(output, "  Log:", strings.Join(info.Plugins.Log, " "))

	fmt.Fprintln(output, " Swarm:", info.Swarm.LocalNodeState)
	printSwarmInfo(output, *info.Info)

	if len(info.Runtimes) > 0 {
		names := make([]string, 0, len(info.Runtimes))
		for name := range info.Runtimes {
			names = append(names, name)
		}
		fmt.Fprintln(output, " Runtimes:", strings.Join(names, " "))
		fmt.Fprintln(output, " Default Runtime:", info.DefaultRuntime)
	}

	if info.OSType == "linux" {
		fmt.Fprintln(output, " Init Binary:", info.InitBinary)

		for _, ci := range []struct {
			Name   string
			Commit types.Commit
		}{
			{"containerd", info.ContainerdCommit},
			{"runc", info.RuncCommit},
			{"init", info.InitCommit},
		} {
			fmt.Fprintf(output, " %s version: %s", ci.Name, ci.Commit.ID)
			if ci.Commit.ID != ci.Commit.Expected {
				fmt.Fprintf(output, " (expected: %s)", ci.Commit.Expected)
			}
			fmt.Fprint(output, "\n")
		}
		if len(info.SecurityOptions) != 0 {
			if kvs, err := types.DecodeSecurityOptions(info.SecurityOptions); err != nil {
				errs = append(errs, err)
			} else {
				fmt.Fprintln(output, " Security Options:")
				for _, so := range kvs {
					fmt.Fprintln(output, "  "+so.Name)
					for _, o := range so.Options {
						switch o.Key {
						case "profile":
							fmt.Fprintln(output, "   Profile:", o.Value)
						}
					}
				}
			}
		}
	}

	// Isolation only has meaning on a Windows daemon.
	if info.OSType == "windows" {
		fmt.Fprintln(output, " Default Isolation:", info.Isolation)
	}

	fprintlnNonEmpty(output, " Kernel Version:", info.KernelVersion)
	fprintlnNonEmpty(output, " Operating System:", info.OperatingSystem)
	fprintlnNonEmpty(output, " OSType:", info.OSType)
	fprintlnNonEmpty(output, " Architecture:", info.Architecture)
	fmt.Fprintln(output, " CPUs:", info.NCPU)
	fmt.Fprintln(output, " Total Memory:", units.BytesSize(float64(info.MemTotal)))
	fprintlnNonEmpty(output, " Name:", info.Name)
	fprintlnNonEmpty(output, " ID:", info.ID)
	fmt.Fprintln(output, " Docker Root Dir:", info.DockerRootDir)
	fmt.Fprintln(output, " Debug Mode:", info.Debug)

	if info.Debug {
		fmt.Fprintln(output, "  File Descriptors:", info.NFd)
		fmt.Fprintln(output, "  Goroutines:", info.NGoroutines)
		fmt.Fprintln(output, "  System Time:", info.SystemTime)
		fmt.Fprintln(output, "  EventsListeners:", info.NEventsListener)
	}

	fprintlnNonEmpty(output, " HTTP Proxy:", info.HTTPProxy)
	fprintlnNonEmpty(output, " HTTPS Proxy:", info.HTTPSProxy)
	fprintlnNonEmpty(output, " No Proxy:", info.NoProxy)
	fprintlnNonEmpty(output, " Username:", info.UserName)
	if len(info.Labels) > 0 {
		fmt.Fprintln(output, " Labels:")
		for _, lbl := range info.Labels {
			fmt.Fprintln(output, "  "+lbl)
		}
	}

	fmt.Fprintln(output, " Experimental:", info.ExperimentalBuild)

	if info.RegistryConfig != nil && (len(info.RegistryConfig.InsecureRegistryCIDRs) > 0 || len(info.RegistryConfig.IndexConfigs) > 0) {
		fmt.Fprintln(output, " Insecure Registries:")
		for _, reg := range info.RegistryConfig.IndexConfigs {
			if !reg.Secure {
				fmt.Fprintln(output, "  "+reg.Name)
			}
		}

		for _, reg := range info.RegistryConfig.InsecureRegistryCIDRs {
			mask, _ := reg.Mask.Size()
			fmt.Fprintf(output, "  %s/%d\n", reg.IP.String(), mask)
		}
	}

	if info.RegistryConfig != nil && len(info.RegistryConfig.Mirrors) > 0 {
		fmt.Fprintln(output, " Registry Mirrors:")
		for _, mirror := range info.RegistryConfig.Mirrors {
			fmt.Fprintln(output, "  "+mirror)
		}
	}

	fmt.Fprintln(output, " Live Restore Enabled:", info.LiveRestoreEnabled)
	if info.ProductLicense != "" {
		fmt.Fprintln(output, " Product License:", info.ProductLicense)
	}

	if info.DefaultAddressPools != nil && len(info.DefaultAddressPools) > 0 {
		fmt.Fprintln(output, " Default Address Pools:")
		for _, pool := range info.DefaultAddressPools {
			fmt.Fprintf(output, "   Base: %s, Size: %d\n", pool.Base, pool.Size)
		}
	}

	fmt.Fprint(output, "\n")
	printServerWarnings(streams.Err(), info)
	return errs
}

//nolint:gocyclo
func printSwarmInfo(output io.Writer, info types.Info) {
	if info.Swarm.LocalNodeState == swarm.LocalNodeStateInactive || info.Swarm.LocalNodeState == swarm.LocalNodeStateLocked {
		return
	}
	fmt.Fprintln(output, "  NodeID:", info.Swarm.NodeID)
	if info.Swarm.Error != "" {
		fmt.Fprintln(output, "  Error:", info.Swarm.Error)
	}
	fmt.Fprintln(output, "  Is Manager:", info.Swarm.ControlAvailable)
	if info.Swarm.Cluster != nil && info.Swarm.ControlAvailable && info.Swarm.Error == "" && info.Swarm.LocalNodeState != swarm.LocalNodeStateError {
		fmt.Fprintln(output, "  ClusterID:", info.Swarm.Cluster.ID)
		fmt.Fprintln(output, "  Managers:", info.Swarm.Managers)
		fmt.Fprintln(output, "  Nodes:", info.Swarm.Nodes)
		var strAddrPool strings.Builder
		if info.Swarm.Cluster.DefaultAddrPool != nil {
			for _, p := range info.Swarm.Cluster.DefaultAddrPool {
				strAddrPool.WriteString(p + "  ")
			}
			fmt.Fprintln(output, "  Default Address Pool:", strAddrPool.String())
			fmt.Fprintln(output, "  SubnetSize:", info.Swarm.Cluster.SubnetSize)
		}
		if info.Swarm.Cluster.DataPathPort > 0 {
			fmt.Fprintln(output, "  Data Path Port:", info.Swarm.Cluster.DataPathPort)
		}
		fmt.Fprintln(output, "  Orchestration:")

		taskHistoryRetentionLimit := int64(0)
		if info.Swarm.Cluster.Spec.Orchestration.TaskHistoryRetentionLimit != nil {
			taskHistoryRetentionLimit = *info.Swarm.Cluster.Spec.Orchestration.TaskHistoryRetentionLimit
		}
		fmt.Fprintln(output, "   Task History Retention Limit:", taskHistoryRetentionLimit)
		fmt.Fprintln(output, "  Raft:")
		fmt.Fprintln(output, "   Snapshot Interval:", info.Swarm.Cluster.Spec.Raft.SnapshotInterval)
		if info.Swarm.Cluster.Spec.Raft.KeepOldSnapshots != nil {
			fmt.Fprintf(output, "   Number of Old Snapshots to Retain: %d\n", *info.Swarm.Cluster.Spec.Raft.KeepOldSnapshots)
		}
		fmt.Fprintln(output, "   Heartbeat Tick:", info.Swarm.Cluster.Spec.Raft.HeartbeatTick)
		fmt.Fprintln(output, "   Election Tick:", info.Swarm.Cluster.Spec.Raft.ElectionTick)
		fmt.Fprintln(output, "  Dispatcher:")
		fmt.Fprintln(output, "   Heartbeat Period:", units.HumanDuration(info.Swarm.Cluster.Spec.Dispatcher.HeartbeatPeriod))
		fmt.Fprintln(output, "  CA Configuration:")
		fmt.Fprintln(output, "   Expiry Duration:", units.HumanDuration(info.Swarm.Cluster.Spec.CAConfig.NodeCertExpiry))
		fmt.Fprintln(output, "   Force Rotate:", info.Swarm.Cluster.Spec.CAConfig.ForceRotate)
		if caCert := strings.TrimSpace(info.Swarm.Cluster.Spec.CAConfig.SigningCACert); caCert != "" {
			fmt.Fprintf(output, "   Signing CA Certificate: \n%s\n\n", caCert)
		}
		if len(info.Swarm.Cluster.Spec.CAConfig.ExternalCAs) > 0 {
			fmt.Fprintln(output, "   External CAs:")
			for _, entry := range info.Swarm.Cluster.Spec.CAConfig.ExternalCAs {
				fmt.Fprintf(output, "     %s: %s\n", entry.Protocol, entry.URL)
			}
		}
		fmt.Fprintln(output, "  Autolock Managers:", info.Swarm.Cluster.Spec.EncryptionConfig.AutoLockManagers)
		fmt.Fprintln(output, "  Root Rotation In Progress:", info.Swarm.Cluster.RootRotationInProgress)
	}
	fmt.Fprintln(output, "  Node Address:", info.Swarm.NodeAddr)
	if len(info.Swarm.RemoteManagers) > 0 {
		managers := []string{}
		for _, entry := range info.Swarm.RemoteManagers {
			managers = append(managers, entry.Addr)
		}
		sort.Strings(managers)
		fmt.Fprintln(output, "  Manager Addresses:")
		for _, entry := range managers {
			fmt.Fprintf(output, "   %s\n", entry)
		}
	}
}

func printServerWarnings(stdErr io.Writer, info *info) {
	if versions.LessThan(info.ClientInfo.APIVersion, "1.42") {
		printSecurityOptionsWarnings(stdErr, *info.Info)
	}
	if len(info.Warnings) > 0 {
		fmt.Fprintln(stdErr, strings.Join(info.Warnings, "\n"))
		return
	}
	// daemon didn't return warnings. Fallback to old behavior
	printServerWarningsLegacy(stdErr, *info.Info)
}

// printSecurityOptionsWarnings prints warnings based on the security options
// returned by the daemon.
// DEPRECATED: warnings are now generated by the daemon, and returned in
// info.Warnings. This function is used to provide backward compatibility with
// daemons that do not provide these warnings. No new warnings should be added
// here.
func printSecurityOptionsWarnings(stdErr io.Writer, info types.Info) {
	if info.OSType == "windows" {
		return
	}
	kvs, _ := types.DecodeSecurityOptions(info.SecurityOptions)
	for _, so := range kvs {
		if so.Name != "seccomp" {
			continue
		}
		for _, o := range so.Options {
			if o.Key == "profile" && o.Value != "default" && o.Value != "builtin" {
				_, _ = fmt.Fprintln(stdErr, "WARNING: You're not using the default seccomp profile")
			}
		}
	}
}

// printServerWarningsLegacy generates warnings based on information returned by the daemon.
// DEPRECATED: warnings are now generated by the daemon, and returned in
// info.Warnings. This function is used to provide backward compatibility with
// daemons that do not provide these warnings. No new warnings should be added
// here.
func printServerWarningsLegacy(stdErr io.Writer, info types.Info) {
	if info.OSType == "windows" {
		return
	}
	if !info.MemoryLimit {
		fmt.Fprintln(stdErr, "WARNING: No memory limit support")
	}
	if !info.SwapLimit {
		fmt.Fprintln(stdErr, "WARNING: No swap limit support")
	}
	if !info.OomKillDisable && info.CgroupVersion != "2" {
		fmt.Fprintln(stdErr, "WARNING: No oom kill disable support")
	}
	if !info.CPUCfsQuota {
		fmt.Fprintln(stdErr, "WARNING: No cpu cfs quota support")
	}
	if !info.CPUCfsPeriod {
		fmt.Fprintln(stdErr, "WARNING: No cpu cfs period support")
	}
	if !info.CPUShares {
		fmt.Fprintln(stdErr, "WARNING: No cpu shares support")
	}
	if !info.CPUSet {
		fmt.Fprintln(stdErr, "WARNING: No cpuset support")
	}
	if !info.IPv4Forwarding {
		fmt.Fprintln(stdErr, "WARNING: IPv4 forwarding is disabled")
	}
	if !info.BridgeNfIptables {
		fmt.Fprintln(stdErr, "WARNING: bridge-nf-call-iptables is disabled")
	}
	if !info.BridgeNfIP6tables {
		fmt.Fprintln(stdErr, "WARNING: bridge-nf-call-ip6tables is disabled")
	}
}

func formatInfo(dockerCli command.Cli, info info, format string) error {
	if format == formatter.JSONFormatKey {
		format = formatter.JSONFormat
	}

	// Ensure slice/array fields render as `[]` not `null`
	if info.ClientInfo != nil && info.ClientInfo.Plugins == nil {
		info.ClientInfo.Plugins = make([]pluginmanager.Plugin, 0)
	}

	tmpl, err := templates.Parse(format)
	if err != nil {
		return cli.StatusError{
			StatusCode: 64,
			Status:     "template parsing error: " + err.Error(),
		}
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
