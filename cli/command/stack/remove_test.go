package stack

import (
	"bytes"
	"errors"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/stretchr/testify/assert"
)

func TestRemoveStack(t *testing.T) {
	allServices := []string{
		objectName("foo", "service1"),
		objectName("foo", "service2"),
		objectName("bar", "service1"),
		objectName("bar", "service2"),
	}
	allServiceIDs := buildObjectIDs(allServices)

	allNetworks := []string{
		objectName("foo", "network1"),
		objectName("bar", "network1"),
	}
	allNetworkIDs := buildObjectIDs(allNetworks)

	allSecrets := []string{
		objectName("foo", "secret1"),
		objectName("foo", "secret2"),
		objectName("bar", "secret1"),
	}
	allSecretIDs := buildObjectIDs(allSecrets)

	allConfigs := []string{
		objectName("foo", "config1"),
		objectName("foo", "config2"),
		objectName("bar", "config1"),
	}
	allConfigIDs := buildObjectIDs(allConfigs)

	// Using API 1.24; removes services, networks, but doesn't remove configs and secrets
	cli := &fakeClient{
		version:  "1.24",
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,
	}
	cmd := newRemoveCommand(test.NewFakeCli(cli, &bytes.Buffer{}))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, allServiceIDs, cli.removedServices)
	assert.Equal(t, allNetworkIDs, cli.removedNetworks)
	assert.Nil(t, cli.removedSecrets)
	assert.Nil(t, cli.removedConfigs)

	// Using API 1.25; removes services, networks, but doesn't remove configs
	cli = &fakeClient{
		version:  "1.25",
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,
	}
	cmd = newRemoveCommand(test.NewFakeCli(cli, &bytes.Buffer{}))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, allServiceIDs, cli.removedServices)
	assert.Equal(t, allNetworkIDs, cli.removedNetworks)
	assert.Equal(t, allSecretIDs, cli.removedSecrets)
	assert.Nil(t, cli.removedConfigs)

	// Using API 1.30; removes services, networks, configs, and secrets
	cli = &fakeClient{
		version:  "1.30",
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,
	}
	cmd = newRemoveCommand(test.NewFakeCli(cli, &bytes.Buffer{}))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, allServiceIDs, cli.removedServices)
	assert.Equal(t, allNetworkIDs, cli.removedNetworks)
	assert.Equal(t, allSecretIDs, cli.removedSecrets)
	assert.Equal(t, allConfigIDs, cli.removedConfigs)
}

func TestRemoveStackSkipEmpty(t *testing.T) {
	out := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	allServices := []string{objectName("bar", "service1"), objectName("bar", "service2")}
	allServiceIDs := buildObjectIDs(allServices)

	allNetworks := []string{objectName("bar", "network1")}
	allNetworkIDs := buildObjectIDs(allNetworks)

	allSecrets := []string{objectName("bar", "secret1")}
	allSecretIDs := buildObjectIDs(allSecrets)

	allConfigs := []string{objectName("bar", "config1")}
	allConfigIDs := buildObjectIDs(allConfigs)

	fakeClient := &fakeClient{
		version:  "1.30",
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,
	}
	fakeCli := test.NewFakeCli(fakeClient, out)
	fakeCli.SetErr(stderr)
	cmd := newRemoveCommand(fakeCli)
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "", out.String())
	assert.Contains(t, stderr.String(), "Nothing found in stack: foo\n")
	assert.Equal(t, allServiceIDs, fakeClient.removedServices)
	assert.Equal(t, allNetworkIDs, fakeClient.removedNetworks)
	assert.Equal(t, allSecretIDs, fakeClient.removedSecrets)
	assert.Equal(t, allConfigIDs, fakeClient.removedConfigs)
}

func TestRemoveContinueAfterError(t *testing.T) {
	allServices := []string{objectName("foo", "service1"), objectName("bar", "service1")}
	allServiceIDs := buildObjectIDs(allServices)

	allNetworks := []string{objectName("foo", "network1"), objectName("bar", "network1")}
	allNetworkIDs := buildObjectIDs(allNetworks)

	allSecrets := []string{objectName("foo", "secret1"), objectName("bar", "secret1")}
	allSecretIDs := buildObjectIDs(allSecrets)

	allConfigs := []string{objectName("foo", "config1"), objectName("bar", "config1")}
	allConfigIDs := buildObjectIDs(allConfigs)

	removedServices := []string{}
	cli := &fakeClient{
		version:  "1.30",
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,

		serviceRemoveFunc: func(serviceID string) error {
			removedServices = append(removedServices, serviceID)

			if strings.Contains(serviceID, "foo") {
				return errors.New("")
			}
			return nil
		},
	}
	cmd := newRemoveCommand(test.NewFakeCli(cli, &bytes.Buffer{}))
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"foo", "bar"})

	assert.EqualError(t, cmd.Execute(), "Failed to remove some resources from stack: foo")
	assert.Equal(t, allServiceIDs, removedServices)
	assert.Equal(t, allNetworkIDs, cli.removedNetworks)
	assert.Equal(t, allSecretIDs, cli.removedSecrets)
	assert.Equal(t, allConfigIDs, cli.removedConfigs)
}
