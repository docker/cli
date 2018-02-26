package kubernetes

import (
	"fmt"
	"io"

	composetypes "github.com/docker/cli/cli/compose/types"
)

func warnUnsupportedFeatures(stderr io.Writer, cfg *composetypes.Config) {
	warnForGlobalNetworks(stderr, cfg)
	for _, s := range cfg.Services {
		warnForServiceNetworks(stderr, s)
		warnForUnsupportedDeploymentStrategy(stderr, s)
		warnForUnsupportedRestartPolicy(stderr, s)
		warnForDeprecatedProperties(stderr, s)
		warnForUnsupportedProperties(stderr, s)
	}
}

func warnForGlobalNetworks(stderr io.Writer, config *composetypes.Config) {
	for network := range config.Networks {
		fmt.Fprintf(stderr, "top-level network %q is ignored\n", network)
	}
}

func warnServicef(stderr io.Writer, service, format string, args ...interface{}) {
	fmt.Fprintf(stderr, "service \"%s\": %s\n", service, fmt.Sprintf(format, args...))
}

func warnForServiceNetworks(stderr io.Writer, s composetypes.ServiceConfig) {
	for network := range s.Networks {
		warnServicef(stderr, s.Name, "network %q is ignored", network)
	}
}

func warnForDeprecatedProperties(stderr io.Writer, s composetypes.ServiceConfig) {
	if s.ContainerName != "" {
		warnServicef(stderr, s.Name, "container_name is deprecated")
	}
	if len(s.Expose) > 0 {
		warnServicef(stderr, s.Name, "expose is deprecated")
	}
}

func warnForUnsupportedDeploymentStrategy(stderr io.Writer, s composetypes.ServiceConfig) {
	config := s.Deploy.UpdateConfig
	if config == nil {
		return
	}
	if config.Delay != 0 {
		warnServicef(stderr, s.Name, "update_config.delay is not supported")
	}
	if config.FailureAction != "" {
		warnServicef(stderr, s.Name, "update_config.failure_action is not supported")
	}
	if config.Monitor != 0 {
		warnServicef(stderr, s.Name, "update_config.monitor is not supported")
	}
	if config.MaxFailureRatio != 0 {
		warnServicef(stderr, s.Name, "update_config.max_failure_ratio is not supported")
	}
}

func warnForUnsupportedRestartPolicy(stderr io.Writer, s composetypes.ServiceConfig) {
	policy := s.Deploy.RestartPolicy
	if policy == nil {
		return
	}

	if policy.Delay != nil {
		warnServicef(stderr, s.Name, "restart_policy.delay is ignored")
	}
	if policy.MaxAttempts != nil {
		warnServicef(stderr, s.Name, "restart_policy.max_attempts is ignored")
	}
	if policy.Window != nil {
		warnServicef(stderr, s.Name, "restart_policy.window is ignored")
	}
}

func warnForUnsupportedProperties(stderr io.Writer, s composetypes.ServiceConfig) { // nolint: gocyclo
	if build := s.Build; build.Context != "" || build.Dockerfile != "" || len(build.Args) > 0 || len(build.Labels) > 0 || len(build.CacheFrom) > 0 || build.Network != "" || build.Target != "" {
		warnServicef(stderr, s.Name, "build is ignored")
	}
	if s.CgroupParent != "" {
		warnServicef(stderr, s.Name, "cgroup_parent is ignored")
	}
	if len(s.Devices) > 0 {
		warnServicef(stderr, s.Name, "devices are ignored")
	}
	if s.DomainName != "" {
		warnServicef(stderr, s.Name, "domainname is ignored")
	}
	if len(s.ExternalLinks) > 0 {
		warnServicef(stderr, s.Name, "external_links are ignored")
	}
	if len(s.Links) > 0 {
		warnServicef(stderr, s.Name, "links are ignored")
	}
	if s.MacAddress != "" {
		warnServicef(stderr, s.Name, "mac_address is ignored")
	}
	if s.NetworkMode != "" {
		warnServicef(stderr, s.Name, "network_mode is ignored")
	}
	if s.Restart != "" {
		warnServicef(stderr, s.Name, "restart is ignored")
	}
	if len(s.SecurityOpt) > 0 {
		warnServicef(stderr, s.Name, "security_opt are ignored")
	}
	if len(s.Ulimits) > 0 {
		warnServicef(stderr, s.Name, "ulimits are ignored")
	}
	if len(s.DependsOn) > 0 {
		warnServicef(stderr, s.Name, "depends_on are ignored")
	}
	if s.CredentialSpec.File != "" {
		warnServicef(stderr, s.Name, "credential_spec is ignored")
	}
	if len(s.DNS) > 0 {
		warnServicef(stderr, s.Name, "dns are ignored")
	}
	if len(s.DNSSearch) > 0 {
		warnServicef(stderr, s.Name, "dns_search are ignored")
	}
	if len(s.EnvFile) > 0 {
		warnServicef(stderr, s.Name, "env_file are ignored")
	}
	if s.StopSignal != "" {
		warnServicef(stderr, s.Name, "stop_signal is ignored")
	}
	if s.Logging != nil {
		warnServicef(stderr, s.Name, "logging is ignored")
	}
	for _, m := range s.Volumes {
		if m.Volume != nil && m.Volume.NoCopy {
			warnServicef(stderr, s.Name, "volume.nocopy is ignored")
		}
		if m.Bind != nil && m.Bind.Propagation != "" {
			warnServicef(stderr, s.Name, "volume.propagation is ignored")
		}
	}
}
