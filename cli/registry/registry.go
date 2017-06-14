package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"golang.org/x/net/context"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/api/types"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	dockerregistry "github.com/docker/docker/registry"
)

// ElectAuthServer returns the default registry to use (by asking the daemon)
func ElectAuthServer(ctx context.Context, c client.APIClient) (string, []error, error) {
	// The daemon `/info` endpoint informs us of the default registry being
	// used. This is essential in cross-platforms environment, where for
	// example a Linux client might be interacting with a Windows daemon, hence
	// the default registry URL might be Windows specific.
	serverAddress := dockerregistry.IndexServer
	var warns []error
	if info, err := c.Info(ctx); err != nil {
		warns = append(warns, fmt.Errorf("failed to get default registry endpoint from daemon (%v). Using system default: %s\n", err, serverAddress))
	} else if info.IndexServerAddress == "" {
		warns = append(warns, fmt.Errorf("Empty registry endpoint from daemon. Using system default: %s\n", serverAddress))
	} else {
		serverAddress = info.IndexServerAddress
	}
	return serverAddress, warns, nil
}

// EncodeAuthToBase64 serializes the auth configuration as JSON base64 payload
func EncodeAuthToBase64(authConfig types.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

// ResolveAuthConfig is like dockerregistry.ResolveAuthConfig, but if using the
// default index, it uses the default index name for the daemon's platform,
// not the client's platform.
func ResolveAuthConfig(ctx context.Context, c client.APIClient, configFile *configfile.ConfigFile, index *registrytypes.IndexInfo) (types.AuthConfig, []error, error) {
	configKey := index.Name
	var (
		warns []error
		err   error
	)
	if index.Official {
		configKey, warns, err = ElectAuthServer(ctx, c)
		if err != nil {
			return types.AuthConfig{}, warns, err
		}
	}

	ac, err := CredentialsStore(configFile, configKey).Get(configKey)
	return ac, warns, err
}
