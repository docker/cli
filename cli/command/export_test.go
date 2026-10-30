package command

import (
	"testing"

	configtypes "github.com/docker/cli/cli/config/types"
)

func SetDesktopAuthConfig(t *testing.T, fn func(registryHostname string) (configtypes.AuthConfig, bool)) {
	t.Helper()
	orig := desktopAuthConfig
	desktopAuthConfig = fn
	t.Cleanup(func() { desktopAuthConfig = orig })
}
