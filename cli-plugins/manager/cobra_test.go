package manager

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func TestPluginResourceAttributesEnvvar(t *testing.T) {
	cmd := &cobra.Command{
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "docker",
		},
	}

	// Ensure basic usage is fine.
	env := appendPluginResourceAttributesEnvvar(nil, cmd, Plugin{Name: "compose"})
	assert.DeepEqual(t, []string{"OTEL_RESOURCE_ATTRIBUTES=docker.cli.cobra.command_path=docker%20compose"}, env)

	// Add a user-based environment variable to OTEL_RESOURCE_ATTRIBUTES.
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "a.b.c=foo")

	env = appendPluginResourceAttributesEnvvar(nil, cmd, Plugin{Name: "compose"})
	assert.DeepEqual(t, []string{"OTEL_RESOURCE_ATTRIBUTES=a.b.c=foo,docker.cli.cobra.command_path=docker%20compose"}, env)
}
