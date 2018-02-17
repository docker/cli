package convert

import (
	"testing"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaceScope(t *testing.T) {
	scoped := Namespace{name: "foo"}.Scope("bar")
	assert.Equal(t, "foo_bar", scoped)
}

func TestAddStackLabel(t *testing.T) {
	labels := map[string]string{
		"something": "labeled",
	}
	actual := AddStackLabel(Namespace{name: "foo"}, labels)
	expected := map[string]string{
		"something":    "labeled",
		LabelNamespace: "foo",
	}
	assert.Equal(t, expected, actual)
}

func TestNetworks(t *testing.T) {
	namespace := Namespace{name: "foo"}
	serviceNetworks := map[string]struct{}{
		"normal":        {},
		"outside":       {},
		"default":       {},
		"attachablenet": {},
	}
	source := networkMap{
		"normal": composetypes.NetworkConfig{
			Driver: "overlay",
			DriverOpts: map[string]string{
				"opt": "value",
			},
			Ipam: composetypes.IPAMConfig{
				Driver: "driver",
				Config: []*composetypes.IPAMPool{
					{
						Subnet: "10.0.0.0",
					},
				},
			},
			Labels: map[string]string{
				"something": "labeled",
			},
		},
		"outside": composetypes.NetworkConfig{
			External: composetypes.External{External: true},
			Name:     "special",
		},
		"attachablenet": composetypes.NetworkConfig{
			Driver:     "overlay",
			Attachable: true,
		},
	}
	expected := map[string]types.NetworkCreate{
		"default": {
			Labels: map[string]string{
				LabelNamespace: "foo",
			},
		},
		"normal": {
			Driver: "overlay",
			IPAM: &network.IPAM{
				Driver: "driver",
				Config: []network.IPAMConfig{
					{
						Subnet: "10.0.0.0",
					},
				},
			},
			Options: map[string]string{
				"opt": "value",
			},
			Labels: map[string]string{
				LabelNamespace: "foo",
				"something":    "labeled",
			},
		},
		"attachablenet": {
			Driver:     "overlay",
			Attachable: true,
			Labels: map[string]string{
				LabelNamespace: "foo",
			},
		},
	}

	networks, externals := Networks(namespace, source, serviceNetworks)
	assert.Equal(t, expected, networks)
	assert.Equal(t, []string{"special"}, externals)
}

func TestSecrets(t *testing.T) {
	namespace := Namespace{name: "foo"}

	secretText := "this is the first secret"
	secretFile := fs.NewFile(t, "convert-secrets", fs.WithContent(secretText))
	defer secretFile.Remove()

	source := map[string]composetypes.SecretConfig{
		"one": {
			File:   secretFile.Path(),
			Labels: map[string]string{"monster": "mash"},
		},
		"ext": {
			External: composetypes.External{
				External: true,
			},
		},
	}

	specs, err := Secrets(namespace, source)
	assert.NoError(t, err)
	require.Len(t, specs, 1)
	secret := specs[0]
	assert.Equal(t, "foo_one", secret.Name)
	assert.Equal(t, map[string]string{
		"monster":      "mash",
		LabelNamespace: "foo",
	}, secret.Labels)
	assert.Equal(t, []byte(secretText), secret.Data)
}

func TestConfigs(t *testing.T) {
	namespace := Namespace{name: "foo"}

	configText := "this is the first config"
	configFile := fs.NewFile(t, "convert-configs", fs.WithContent(configText))
	defer configFile.Remove()

	source := map[string]composetypes.ConfigObjConfig{
		"one": {
			File:   configFile.Path(),
			Labels: map[string]string{"monster": "mash"},
		},
		"ext": {
			External: composetypes.External{
				External: true,
			},
		},
	}

	specs, err := Configs(namespace, source)
	assert.NoError(t, err)
	require.Len(t, specs, 1)
	config := specs[0]
	assert.Equal(t, "foo_one", config.Name)
	assert.Equal(t, map[string]string{
		"monster":      "mash",
		LabelNamespace: "foo",
	}, config.Labels)
	assert.Equal(t, []byte(configText), config.Data)
}
