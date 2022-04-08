package environment

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"gotest.tools/v3/icmd"
	"gotest.tools/v3/poll"
	"gotest.tools/v3/skip"
)

// Setup a new environment
func Setup() error {
	dockerHost := os.Getenv("TEST_DOCKER_HOST")
	if dockerHost == "" {
		return errors.New("$TEST_DOCKER_HOST must be set")
	}
	if err := os.Setenv(client.EnvOverrideHost, dockerHost); err != nil {
		return err
	}

	if dockerCertPath := os.Getenv("TEST_DOCKER_CERT_PATH"); dockerCertPath != "" {
		if err := os.Setenv(client.EnvOverrideCertPath, dockerCertPath); err != nil {
			return err
		}
		if err := os.Setenv(client.EnvTLSVerify, "1"); err != nil {
			return err
		}
	}

	if val := boolFromString(os.Getenv("TEST_REMOTE_DAEMON")); val {
		if err := os.Setenv("REMOTE_DAEMON", "1"); err != nil {
			return err
		}
	}

	if val := boolFromString(os.Getenv("TEST_SKIP_PLUGIN_TESTS")); val {
		if err := os.Setenv("SKIP_PLUGIN_TESTS", "1"); err != nil {
			return err
		}
	}

	return nil
}

// RemoteDaemon returns true if running against a remote daemon
func RemoteDaemon() bool {
	return os.Getenv("REMOTE_DAEMON") != ""
}

// SkipPluginTests returns if plugin tests should be skipped
func SkipPluginTests() bool {
	return os.Getenv("SKIP_PLUGIN_TESTS") != ""
}

// boolFromString determines boolean value from string
func boolFromString(val string) bool {
	switch strings.ToLower(val) {
	case "true", "1":
		return true
	default:
		return false
	}
}

// DefaultPollSettings used with gotestyourself/poll
var DefaultPollSettings = poll.WithDelay(100 * time.Millisecond)

// SkipIfNotExperimentalDaemon returns whether the test docker daemon is in experimental mode
func SkipIfNotExperimentalDaemon(t *testing.T) {
	t.Helper()
	result := icmd.RunCmd(icmd.Command("docker", "info", "--format", "{{.ExperimentalBuild}}"))
	result.Assert(t, icmd.Expected{Err: icmd.None})
	experimentalBuild := strings.TrimSpace(result.Stdout()) == "true"
	skip.If(t, !experimentalBuild, "running against a non-experimental daemon")
}

// SkipIfDaemonNotLinux skips the test unless the running docker daemon is on Linux
func SkipIfDaemonNotLinux(t *testing.T) {
	t.Helper()
	result := icmd.RunCmd(icmd.Command("docker", "info", "--format", "{{.OSType}}"))
	result.Assert(t, icmd.Expected{Err: icmd.None})
	isLinux := strings.TrimSpace(result.Stdout()) == "linux"
	skip.If(t, !isLinux, "running against a Linux daemon")
}

// SkipIfCgroupNamespacesNotSupported skips the test if the running docker daemon doesn't support cgroup namespaces
func SkipIfCgroupNamespacesNotSupported(t *testing.T) {
	t.Helper()
	result := icmd.RunCmd(icmd.Command("docker", "info", "--format", "{{.SecurityOptions}}"))
	result.Assert(t, icmd.Expected{Err: icmd.None})
	cgroupNsFound := strings.Contains(result.Stdout(), "name=cgroupns")

	skip.If(t, !cgroupNsFound, fmt.Sprintf("running against a daemon that doesn't support cgroup namespaces (security options: %s)", result.Stdout()))
}
