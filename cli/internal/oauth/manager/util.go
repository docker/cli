package manager

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/cli/cli/version"
)

const (
	audience = "https://hub.docker.com"
	tenant   = "login.docker.com"
	clientID = "DHWuMefQ1v4lxENpz8oUYH50yYSwyPvi"
)

func NewManager(store credentials.Store) (*OAuthManager, error) {
	cliVersion := strings.ReplaceAll(version.Version, ".", "_")
	options := OAuthManagerOptions{
		Audience:   audience,
		ClientID:   clientID,
		Tenant:     tenant,
		DeviceName: "docker-cli:" + cliVersion,
		Store:      store,
	}

	options.DeviceName = fmt.Sprintf("docker-cli:%s:%s-%s", cliVersion, runtime.GOOS, runtime.GOARCH)

	authManager, err := New(options)
	if err != nil {
		return nil, err
	}
	return authManager, nil
}
