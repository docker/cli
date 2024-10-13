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

func TestCompleteFileNames(t *testing.T) {
	values, directives := FileNames(nil, nil, "")
	assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveDefault))
	assert.Check(t, is.Len(values, 0))
}

func TestCompleteFromList(t *testing.T) {
	expected := []string{"one", "two", "three"}

	values, directives := FromList(expected...)(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, is.DeepEqual(values, expected))
}

func TestCompleteNoComplete(t *testing.T) {
	values, directives := NoComplete(nil, nil, "")
	assert.Check(t, is.Equal(directives, cobra.ShellCompDirectiveNoFileComp))
	assert.Check(t, is.Len(values, 0))
}

func TestCompletePlatforms(t *testing.T) {
	values, directives := Platforms(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, is.DeepEqual(values, commonPlatforms))
}
