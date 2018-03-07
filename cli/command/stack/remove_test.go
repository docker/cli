package stack

import (
	"errors"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/cli/internal/test"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
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

	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.services), client.removedServices))
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.networks), client.removedNetworks))
	assert.Check(t, is.Len(client.removedSecrets, 0))
	assert.Check(t, is.Len(client.removedConfigs, 0))
}

func TestRemoveStackVersion125DoesNotRemoveConfigs(t *testing.T) {
	client := fakeClientForRemoveStackTest("1.25")
	cmd := newRemoveCommand(test.NewFakeCli(client))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.services), client.removedServices))
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.networks), client.removedNetworks))
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.secrets), client.removedSecrets))
	assert.Check(t, is.Len(client.removedConfigs, 0))
}

func TestRemoveStackVersion130RemovesEverything(t *testing.T) {
	client := fakeClientForRemoveStackTest("1.30")
	cmd := newRemoveCommand(test.NewFakeCli(client))
	cmd.SetArgs([]string{"foo", "bar"})

	assert.NilError(t, cmd.Execute())
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.services), client.removedServices))
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.networks), client.removedNetworks))
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.secrets), client.removedSecrets))
	assert.Check(t, is.DeepEqual(buildObjectIDs(client.configs), client.removedConfigs))
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

	assert.NilError(t, cmd.Execute())
	expectedList := []string{"Removing service bar_service1",
		"Removing service bar_service2",
		"Removing secret bar_secret1",
		"Removing config bar_config1",
		"Removing network bar_network1\n",
	}
	assert.Check(t, is.Equal(strings.Join(expectedList, "\n"), fakeCli.OutBuffer().String()))
	assert.Check(t, is.Contains(fakeCli.ErrBuffer().String(), "Nothing found in stack: foo\n"))
	assert.Check(t, is.DeepEqual(allServiceIDs, fakeClient.removedServices))
	assert.Check(t, is.DeepEqual(allNetworkIDs, fakeClient.removedNetworks))
	assert.Check(t, is.DeepEqual(allSecretIDs, fakeClient.removedSecrets))
	assert.Check(t, is.DeepEqual(allConfigIDs, fakeClient.removedConfigs))
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

	assert.Error(t, cmd.Execute(), "Failed to remove some resources from stack: foo")
	assert.Check(t, is.DeepEqual(allServiceIDs, removedServices))
	assert.Check(t, is.DeepEqual(allNetworkIDs, cli.removedNetworks))
	assert.Check(t, is.DeepEqual(allSecretIDs, cli.removedSecrets))
	assert.Check(t, is.DeepEqual(allConfigIDs, cli.removedConfigs))
}
