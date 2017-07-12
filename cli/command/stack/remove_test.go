package stack

import (
	"errors"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/cli/internal/test"
	"github.com/stretchr/testify/assert"
)

func fakeClientForRemoveStackTest(version string) *fakeClient {
	allServices := []string{
		objectName("foo", "service1"),
		objectName("foo", "service2"),
		objectName("bar", "service1"),
		objectName("bar", "service2"),
	}
	allNetworks := []string{
		objectName("foo", "network1"),
		objectName("bar", "network1"),
	}
	allSecrets := []string{
		objectName("foo", "secret1"),
		objectName("foo", "secret2"),
		objectName("bar", "secret1"),
	}
	allConfigs := []string{
		objectName("foo", "config1"),
		objectName("foo", "config2"),
		objectName("bar", "config1"),
	}
	return &fakeClient{
		version:  version,
		services: allServices,
		networks: allNetworks,
		secrets:  allSecrets,
		configs:  allConfigs,
	}
}

func TestRemoveStackVersion124DoesNotRemoveConfigsOrSecrets(t *testing.T) {
	client := fakeClientForRemoveStackTest("1.24")
	cmd := newRemoveCommand(test.NewFakeCli(client))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, buildObjectIDs(client.services), client.removedServices)
	assert.Equal(t, buildObjectIDs(client.networks), client.removedNetworks)
	assert.Nil(t, client.removedSecrets)
	assert.Nil(t, client.removedConfigs)
}

func TestRemoveStackVersion125DoesNotRemoveConfigs(t *testing.T) {
	client := fakeClientForRemoveStackTest("1.25")
	cmd := newRemoveCommand(test.NewFakeCli(client))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, buildObjectIDs(client.services), client.removedServices)
	assert.Equal(t, buildObjectIDs(client.networks), client.removedNetworks)
	assert.Equal(t, buildObjectIDs(client.secrets), client.removedSecrets)
	assert.Nil(t, client.removedConfigs)
}

func TestRemoveStackVersion130RemovesEverything(t *testing.T) {
	client := fakeClientForRemoveStackTest("1.30")
	cmd := newRemoveCommand(test.NewFakeCli(client))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, buildObjectIDs(client.services), client.removedServices)
	assert.Equal(t, buildObjectIDs(client.networks), client.removedNetworks)
	assert.Equal(t, buildObjectIDs(client.secrets), client.removedSecrets)
	assert.Equal(t, buildObjectIDs(client.configs), client.removedConfigs)
}

func TestRemoveStackSkipEmpty(t *testing.T) {
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
	fakeCli := test.NewFakeCli(fakeClient)
	cmd := newRemoveCommand(fakeCli)
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NoError(t, cmd.Execute())
	assert.Equal(t, "", fakeCli.OutBuffer().String())
	assert.Contains(t, fakeCli.ErrBuffer().String(), "Nothing found in stack: foo\n")
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
	cmd := newRemoveCommand(test.NewFakeCli(cli))
	cmd.SetOutput(ioutil.Discard)
	cmd.SetArgs([]string{"foo", "bar"})

	assert.EqualError(t, cmd.Execute(), "Failed to remove some resources from stack: foo")
	assert.Equal(t, allServiceIDs, removedServices)
	assert.Equal(t, allNetworkIDs, cli.removedNetworks)
	assert.Equal(t, allSecretIDs, cli.removedSecrets)
	assert.Equal(t, allConfigIDs, cli.removedConfigs)
}
