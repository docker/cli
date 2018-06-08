package environment

import (
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gotest.tools/poll"
)

// Setup a new environment
func Setup() error {
	dockerHost := os.Getenv("TEST_DOCKER_HOST")
	if dockerHost == "" {
		return errors.New("$TEST_DOCKER_HOST must be set")
	}
	if err := os.Setenv("DOCKER_HOST", dockerHost); err != nil {
		return err
	}

	if dockerCertPath := os.Getenv("TEST_DOCKER_CERT_PATH"); dockerCertPath != "" {
		if err := os.Setenv("DOCKER_CERT_PATH", dockerCertPath); err != nil {
			return err
		}
		if err := os.Setenv("DOCKER_TLS_VERIFY", "1"); err != nil {
			return err
		}
	}

	if kubeConfig := os.Getenv("TEST_KUBECONFIG"); kubeConfig != "" {
		if err := os.Setenv("KUBECONFIG", kubeConfig); err != nil {
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

// KubernetesEnabled returns if Kubernetes testing is enabled
func KubernetesEnabled() bool {
	return os.Getenv("KUBECONFIG") != ""
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
