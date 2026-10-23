package registryauth

import (
	"os"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/config/configfile"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/internal/secretsengine"
	"github.com/moby/moby/api/pkg/authconfig"
	registrytypes "github.com/moby/moby/api/types/registry"
)

type configProvider interface {
	ConfigFile() *configfile.ConfigFile
}

type authProvider interface {
	RegistryAuthProvider() secretsengine.AuthProvider
}

func EncodedAuth(dockerCLI configProvider, image string) (string, error) {
	if p, ok := dockerCLI.(authProvider); ok {
		return encodedAuth(p.RegistryAuthProvider(), image)
	}
	return encodedAuth(secretsengine.NewDesktop(dockerCLI.ConfigFile()), image)
}

func encodedAuth(auth secretsengine.AuthProvider, image string) (string, error) {
	registryRef, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return "", err
	}
	domain := reference.Domain(registryRef)
	authConfig, ok := envAuthConfig(domain)
	if !ok {
		authConfig, err = auth.GetAuthConfig(domain)
		if err != nil {
			return "", err
		}
	}

	return authconfig.Encode(registrytypes.AuthConfig{
		Username:      authConfig.Username,
		Password:      authConfig.Password,
		ServerAddress: authConfig.ServerAddress,
		Auth:          authConfig.Auth,
		IdentityToken: authConfig.IdentityToken,
		RegistryToken: authConfig.RegistryToken,
	})
}

func envAuthConfig(registryHostname string) (configtypes.AuthConfig, bool) {
	if os.Getenv(configfile.DockerEnvConfigKey) == "" {
		return configtypes.AuthConfig{}, false
	}
	authConfig, err := configfile.New("").GetAuthConfig(registryHostname)
	return authConfig, err == nil && authConfig.Username != ""
}
