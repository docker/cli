package completion

import (
	"sort"
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/env"
)

func TestCompleteEnvVarNames(t *testing.T) {
	env.PatchAll(t, map[string]string{
		"ENV_A": "hello-a",
		"ENV_B": "hello-b",
	})
	values, directives := EnvVarNames(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")

	sort.Strings(values)
	expected := []string{"ENV_A", "ENV_B"}
	assert.Check(t, is.DeepEqual(values, expected))
}

func TestCompletePlatforms(t *testing.T) {
	values, directives := Platforms(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, is.DeepEqual(values, commonPlatforms))
}
