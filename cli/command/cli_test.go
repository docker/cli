package command

import (
	"os"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIClientFromFlags(t *testing.T) {
	host := "unix://path"
	opts := &flags.CommonOptions{Hosts: []string{host}}
	configFile := &configfile.ConfigFile{
		HTTPHeaders: map[string]string{
			"My-Header": "Custom-Value",
		},
	}
	apiclient, err := NewAPIClientFromFlags(opts, configFile)
	require.NoError(t, err)
	assert.Equal(t, host, apiclient.DaemonHost())

	expectedHeaders := map[string]string{
		"My-Header":  "Custom-Value",
		"User-Agent": UserAgent(),
	}
	assert.Equal(t, expectedHeaders, apiclient.(*client.Client).CustomHTTPHeaders())
	assert.Equal(t, api.DefaultVersion, apiclient.ClientVersion())
}

func TestNewAPIClientFromFlagsWithAPIVersionFromEnv(t *testing.T) {
	customVersion := "v3.3.3"
	defer patchEnvVariable(t, "DOCKER_API_VERSION", customVersion)()

	opts := &flags.CommonOptions{}
	configFile := &configfile.ConfigFile{}
	apiclient, err := NewAPIClientFromFlags(opts, configFile)
	require.NoError(t, err)
	assert.Equal(t, customVersion, apiclient.ClientVersion())
}

// TODO: move to gotestyourself
func patchEnvVariable(t *testing.T, key, value string) func() {
	oldValue, ok := os.LookupEnv(key)
	require.NoError(t, os.Setenv(key, value))
	return func() {
		if !ok {
			require.NoError(t, os.Unsetenv(key))
			return
		}
		require.NoError(t, os.Setenv(key, oldValue))
	}
}
