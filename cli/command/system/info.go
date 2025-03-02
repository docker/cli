// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package system

import (
	"context"
	"errors"
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
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/system"
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

type dockerInfo struct {
	// This field should/could be ServerInfo but is anonymous to
	// preserve backwards compatibility in the JSON rendering
	// which has ServerInfo immediately within the top-level
	// object.
	*system.Info `json:",omitempty"`
	ServerErrors []string `json:",omitempty"`
	UserName     string   `json:"-"`

	ClientInfo   *clientInfo `json:",omitempty"`
	ClientErrors []string    `json:",omitempty"`
}

func (i *dockerInfo) clientPlatform() string {
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
			return runInfo(cmd.Context(), cmd, dockerCli, &opts)
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

func runInfo(ctx context.Context, cmd *cobra.Command, dockerCli command.Cli, opts *infoOptions) error {
	info := dockerInfo{
		ClientInfo: &clientInfo{
			// Don't pass a dockerCLI to newClientVersion(), because we currently
			// don't include negotiated API version, and want to avoid making an
			// API connection when only printing the Client section.
			clientVersion: newClientVersion(dockerCli.CurrentContext(), nil),
			Debug:         debug.IsEnabled(),
		},
		Info: &system.Info{},
	}
	if plugins, err := pluginmanager.ListPlugins(dockerCli, cmd.Root()); err == nil {
		info.ClientInfo.Plugins = plugins
	} else {
		info.ClientErrors = append(info.ClientErrors, err.Error())
	}

	if needsServerInfo(opts.format, info) {
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
				fprintln(dockerCli.Err(), err)
			}
		}
	}

	if opts.format == "" {
		info.UserName = dockerCli.ConfigFile().AuthConfigs[registry.IndexServer].Username
		info.ClientInfo.APIVersion = dockerCli.CurrentVersion()
		return prettyPrintInfo(dockerCli, info)
	}
	return formatInfo(dockerCli.Out(), info, opts.format)
}

// placeHolders does a rudimentary match for possible placeholders in a
// template, matching a '.', followed by an letter (a-z/A-Z).
var placeHolders = regexp.MustCompile(`\.[a-zA-Z]`)

// needsServerInfo detects if the given template uses any server information.
// If only client-side information is used in the template, we can skip
// connecting to the daemon. This allows (e.g.) to only get cli-plugin
// information, without also making a (potentially expensive) API call.
func needsServerInfo(template string, info dockerInfo) bool {
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

func prettyPrintInfo(streams command.Streams, info dockerInfo) error {
	// Only append the platform info if it's not empty, to prevent printing a trailing space.
	if p := info.clientPlatform(); p != "" {
		fprintln(streams.Out(), "Client:", p)
	} else {
		fprintln(streams.Out(), "Client:")
	}
	if info.ClientInfo != nil {
		prettyPrintClientInfo(streams, *info.ClientInfo)
	}
	for _, err := range info.ClientErrors {
		fprintln(streams.Err(), "ERROR:", err)
	}

	fprintln(streams.Out())
	fprintln(streams.Out(), "Server:")
	if info.Info != nil {
		for _, err := range prettyPrintServerInfo(streams, &info) {
			info.ServerErrors = append(info.ServerErrors, err.Error())
		}
	}
	for _, err := range info.ServerErrors {
		fprintln(streams.Err(), "ERROR:", err)
	}

	if len(info.ServerErrors) > 0 || len(info.ClientErrors) > 0 {
		return errors.New("errors pretty printing info")
	}
	return nil
}

func prettyPrintClientInfo(streams command.Streams, info clientInfo) {
	fprintlnNonEmpty(streams.Out(), " Version:   ", info.Version)
	fprintln(streams.Out(), " Context:   ", info.Context)
	fprintln(streams.Out(), " Debug Mode:", info.Debug)

	if len(info.Plugins) > 0 {
		fprintln(streams.Out(), " Plugins:")
		for _, p := range info.Plugins {
			if p.Err == nil {
				fprintf(streams.Out(), "  %s: %s (%s)\n", p.Name, p.ShortDescription, p.Vendor)
				fprintlnNonEmpty(streams.Out(), "    Version: ", p.Version)
				fprintlnNonEmpty(streams.Out(), "    Path:    ", p.Path)
			} else {
				info.Warnings = append(info.Warnings, fmt.Sprintf("WARNING: Plugin %q is not valid: %s", p.Path, p.Err))
			}
		}
	}

	if len(info.Warnings) > 0 {
		fprintln(streams.Err(), strings.Join(info.Warnings, "\n"))
	}
}

//nolint:gocyclo
func prettyPrintServerInfo(streams command.Streams, info *dockerInfo) []error {
	var errs []error
	output := streams.Out()

	fprintln(output, " Containers:", info.Containers)
	fprintln(output, "  Running:", info.ContainersRunning)
	fprintln(output, "  Paused:", info.ContainersPaused)
	fprintln(output, "  Stopped:", info.ContainersStopped)
	fprintln(output, " Images:", info.Images)
	fprintlnNonEmpty(output, " Server Version:", info.ServerVersion)
	fprintlnNonEmpty(output, " Storage Driver:", info.Driver)
	if info.DriverStatus != nil {
		for _, pair := range info.DriverStatus {
			fprintf(output, "  %s: %s\n", pair[0], pair[1])
		}
	}
	if info.SystemStatus != nil {
		for _, pair := range info.SystemStatus {
			fprintf(output, " %s: %s\n", pair[0], pair[1])
		}
	}
	fprintlnNonEmpty(output, " Logging Driver:", info.LoggingDriver)
	fprintlnNonEmpty(output, " Cgroup Driver:", info.CgroupDriver)
	fprintlnNonEmpty(output, " Cgroup Version:", info.CgroupVersion)

	fprintln(output, " Plugins:")
	fprintln(output, "  Volume:", strings.Join(info.Plugins.Volume, " "))
	fprintln(output, "  Network:", strings.Join(info.Plugins.Network, " "))

	if len(info.Plugins.Authorization) != 0 {
		fprintln(output, "  Authorization:", strings.Join(info.Plugins.Authorization, " "))
	}

	fprintln(output, "  Log:", strings.Join(info.Plugins.Log, " "))

	if len(info.CDISpecDirs) > 0 {
		fprintln(output, " CDI spec directories:")
		for _, dir := range info.CDISpecDirs {
			fprintf(output, "  %s\n", dir)
		}
	}

	fprintln(output, " Swarm:", info.Swarm.LocalNodeState)
	printSwarmInfo(output, *info.Info)

	if len(info.Runtimes) > 0 {
		names := make([]string, 0, len(info.Runtimes))
		for name := range info.Runtimes {
			names = append(names, name)
		}
		fprintln(output, " Runtimes:", strings.Join(names, " "))
		fprintln(output, " Default Runtime:", info.DefaultRuntime)
	}

	if info.OSType == "linux" {
		fprintln(output, " Init Binary:", info.InitBinary)
		fprintln(output, " containerd version:", info.ContainerdCommit.ID)
		fprintln(output, " runc version:", info.RuncCommit.ID)
		fprintln(output, " init version:", info.InitCommit.ID)
		if len(info.SecurityOptions) != 0 {
			if kvs, err := system.DecodeSecurityOptions(info.SecurityOptions); err != nil {
				errs = append(errs, err)
			} else {
				fprintln(output, " Security Options:")
				for _, so := range kvs {
					fprintln(output, "  "+so.Name)
					for _, o := range so.Options {
						if o.Key == "profile" {
							fprintln(output, "   Profile:", o.Value)
						}
					}
				}
			}
		}
	}

	// Isolation only has meaning on a Windows daemon.
	if info.OSType == "windows" {
		fprintln(output, " Default Isolation:", info.Isolation)
	}

	fprintlnNonEmpty(output, " Kernel Version:", info.KernelVersion)
	fprintlnNonEmpty(output, " Operating System:", info.OperatingSystem)
	fprintlnNonEmpty(output, " OSType:", info.OSType)
	fprintlnNonEmpty(output, " Architecture:", info.Architecture)
	fprintln(output, " CPUs:", info.NCPU)
	fprintln(output, " Total Memory:", units.BytesSize(float64(info.MemTotal)))
	fprintlnNonEmpty(output, " Name:", info.Name)
	fprintlnNonEmpty(output, " ID:", info.ID)
	fprintln(output, " Docker Root Dir:", info.DockerRootDir)
	fprintln(output, " Debug Mode:", info.Debug)

	// The daemon collects this information regardless if "debug" is
	// enabled. Print the debugging information if either the daemon,
	// or the client has debug enabled. We should probably improve this
	// logic and print any of these if set (but some special rules are
	// needed for file-descriptors, which may use "-1".
	if info.Debug || debug.IsEnabled() {
		fprintln(output, "  File Descriptors:", info.NFd)
		fprintln(output, "  Goroutines:", info.NGoroutines)
		fprintln(output, "  System Time:", info.SystemTime)
		fprintln(output, "  EventsListeners:", info.NEventsListener)
	}

	fprintlnNonEmpty(output, " HTTP Proxy:", info.HTTPProxy)
	fprintlnNonEmpty(output, " HTTPS Proxy:", info.HTTPSProxy)
	fprintlnNonEmpty(output, " No Proxy:", info.NoProxy)
	fprintlnNonEmpty(output, " Username:", info.UserName)
	if len(info.Labels) > 0 {
		fprintln(output, " Labels:")
		for _, lbl := range info.Labels {
			fprintln(output, "  "+lbl)
		}
	}

	fprintln(output, " Experimental:", info.ExperimentalBuild)

	if info.RegistryConfig != nil && (len(info.RegistryConfig.InsecureRegistryCIDRs) > 0 || len(info.RegistryConfig.IndexConfigs) > 0) {
		fprintln(output, " Insecure Registries:")
		for _, registryConfig := range info.RegistryConfig.IndexConfigs {
			if !registryConfig.Secure {
				fprintln(output, "  "+registryConfig.Name)
			}
		}

		for _, registryConfig := range info.RegistryConfig.InsecureRegistryCIDRs {
			mask, _ := registryConfig.Mask.Size()
			fprintf(output, "  %s/%d\n", registryConfig.IP.String(), mask)
		}
	}

	if info.RegistryConfig != nil && len(info.RegistryConfig.Mirrors) > 0 {
		fprintln(output, " Registry Mirrors:")
		for _, mirror := range info.RegistryConfig.Mirrors {
			fprintln(output, "  "+mirror)
		}
	}

	fprintln(output, " Live Restore Enabled:", info.LiveRestoreEnabled)
	if info.ProductLicense != "" {
		fprintln(output, " Product License:", info.ProductLicense)
	}

	if len(info.DefaultAddressPools) > 0 {
		fprintln(output, " Default Address Pools:")
		for _, pool := range info.DefaultAddressPools {
			fprintf(output, "   Base: %s, Size: %d\n", pool.Base, pool.Size)
		}
	}

	fprintln(output)
	for _, w := range info.Warnings {
		fprintln(streams.Err(), w)
	}

	return errs
}

//nolint:gocyclo
func printSwarmInfo(output io.Writer, info system.Info) {
	if info.Swarm.LocalNodeState == swarm.LocalNodeStateInactive || info.Swarm.LocalNodeState == swarm.LocalNodeStateLocked {
		return
	}
	fprintln(output, "  NodeID:", info.Swarm.NodeID)
	if info.Swarm.Error != "" {
		fprintln(output, "  Error:", info.Swarm.Error)
	}
	fprintln(output, "  Is Manager:", info.Swarm.ControlAvailable)
	if info.Swarm.Cluster != nil && info.Swarm.ControlAvailable && info.Swarm.Error == "" && info.Swarm.LocalNodeState != swarm.LocalNodeStateError {
		fprintln(output, "  ClusterID:", info.Swarm.Cluster.ID)
		fprintln(output, "  Managers:", info.Swarm.Managers)
		fprintln(output, "  Nodes:", info.Swarm.Nodes)
		var strAddrPool strings.Builder
		if info.Swarm.Cluster.DefaultAddrPool != nil {
			for _, p := range info.Swarm.Cluster.DefaultAddrPool {
				strAddrPool.WriteString(p + "  ")
			}
			fprintln(output, "  Default Address Pool:", strAddrPool.String())
			fprintln(output, "  SubnetSize:", info.Swarm.Cluster.SubnetSize)
		}
		if info.Swarm.Cluster.DataPathPort > 0 {
			fprintln(output, "  Data Path Port:", info.Swarm.Cluster.DataPathPort)
		}
		fprintln(output, "  Orchestration:")

		taskHistoryRetentionLimit := int64(0)
		if info.Swarm.Cluster.Spec.Orchestration.TaskHistoryRetentionLimit != nil {
			taskHistoryRetentionLimit = *info.Swarm.Cluster.Spec.Orchestration.TaskHistoryRetentionLimit
		}
		fprintln(output, "   Task History Retention Limit:", taskHistoryRetentionLimit)
		fprintln(output, "  Raft:")
		fprintln(output, "   Snapshot Interval:", info.Swarm.Cluster.Spec.Raft.SnapshotInterval)
		if info.Swarm.Cluster.Spec.Raft.KeepOldSnapshots != nil {
			fprintf(output, "   Number of Old Snapshots to Retain: %d\n", *info.Swarm.Cluster.Spec.Raft.KeepOldSnapshots)
		}
		fprintln(output, "   Heartbeat Tick:", info.Swarm.Cluster.Spec.Raft.HeartbeatTick)
		fprintln(output, "   Election Tick:", info.Swarm.Cluster.Spec.Raft.ElectionTick)
		fprintln(output, "  Dispatcher:")
		fprintln(output, "   Heartbeat Period:", units.HumanDuration(info.Swarm.Cluster.Spec.Dispatcher.HeartbeatPeriod))
		fprintln(output, "  CA Configuration:")
		fprintln(output, "   Expiry Duration:", units.HumanDuration(info.Swarm.Cluster.Spec.CAConfig.NodeCertExpiry))
		fprintln(output, "   Force Rotate:", info.Swarm.Cluster.Spec.CAConfig.ForceRotate)
		if caCert := strings.TrimSpace(info.Swarm.Cluster.Spec.CAConfig.SigningCACert); caCert != "" {
			fprintf(output, "   Signing CA Certificate: \n%s\n\n", caCert)
		}
		if len(info.Swarm.Cluster.Spec.CAConfig.ExternalCAs) > 0 {
			fprintln(output, "   External CAs:")
			for _, entry := range info.Swarm.Cluster.Spec.CAConfig.ExternalCAs {
				fprintf(output, "     %s: %s\n", entry.Protocol, entry.URL)
			}
		}
		fprintln(output, "  Autolock Managers:", info.Swarm.Cluster.Spec.EncryptionConfig.AutoLockManagers)
		fprintln(output, "  Root Rotation In Progress:", info.Swarm.Cluster.RootRotationInProgress)
	}
	fprintln(output, "  Node Address:", info.Swarm.NodeAddr)
	if len(info.Swarm.RemoteManagers) > 0 {
		managers := []string{}
		for _, entry := range info.Swarm.RemoteManagers {
			managers = append(managers, entry.Addr)
		}
		sort.Strings(managers)
		fprintln(output, "  Manager Addresses:")
		for _, entry := range managers {
			fprintf(output, "   %s\n", entry)
		}
	}
}

func formatInfo(output io.Writer, info dockerInfo, format string) error {
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
	err = tmpl.Execute(output, info)
	fprintln(output)
	return err
}

func fprintf(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

func fprintln(w io.Writer, a ...any) {
	_, _ = fmt.Fprintln(w, a...)
}

func fprintlnNonEmpty(w io.Writer, label, value string) {
	if value != "" {
		_, _ = fmt.Fprintln(w, label, value)
	}
}
