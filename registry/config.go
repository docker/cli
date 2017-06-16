package registry

import (
	"github.com/docker/cli/config/configfile"
	"github.com/docker/cli/config/credentials"
	"github.com/docker/docker/api/types"
)

// GetAllCredentials returns all of the credentials stored in all of the
// configured credential stores.
func GetAllCredentials(configFile *configfile.ConfigFile) (map[string]types.AuthConfig, error) {
	auths := make(map[string]types.AuthConfig)
	for registry := range configFile.CredentialHelpers {
		helper := CredentialsStore(configFile, registry)
		newAuths, err := helper.GetAll()
		if err != nil {
			return nil, err
		}
		addAll(auths, newAuths)
	}
	defaultStore := CredentialsStore(configFile, "")
	newAuths, err := defaultStore.GetAll()
	if err != nil {
		return nil, err
	}
	addAll(auths, newAuths)
	return auths, nil
}

func addAll(to, from map[string]types.AuthConfig) {
	for reg, ac := range from {
		to[reg] = ac
	}
}

// CredentialsStore returns a new credentials store based
// on the settings provided in the configuration file. Empty string returns
// the default credential store.
func CredentialsStore(configFile *configfile.ConfigFile, serverAddress string) credentials.Store {
	if helper := getConfiguredCredentialStore(configFile, serverAddress); helper != "" {
		return credentials.NewNativeStore(configFile, helper)
	}
	return credentials.NewFileStore(configFile)
}

// getConfiguredCredentialStore returns the credential helper configured for the
// given registry, the default credsStore, or the empty string if neither are
// configured.
func getConfiguredCredentialStore(c *configfile.ConfigFile, serverAddress string) string {
	if c.CredentialHelpers != nil && serverAddress != "" {
		if helper, exists := c.CredentialHelpers[serverAddress]; exists {
			return helper
		}
	}
	return c.CredentialsStore
}
