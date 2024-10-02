package container

import (
	"strings"
	"testing"

	"github.com/moby/sys/signal"
	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCompleteLinuxCapabilityNames(t *testing.T) {
	names, directives := completeLinuxCapabilityNames(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Assert(t, len(names) > 1)
	assert.Check(t, names[0] == allCaps)
	for _, name := range names[1:] {
		assert.Check(t, strings.HasPrefix(name, "CAP_"))
		assert.Check(t, is.Equal(name, strings.ToUpper(name)), "Should be formatted uppercase")
	}
}

func TestCompleteRestartPolicies(t *testing.T) {
	values, directives := completeRestartPolicies(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	expected := restartPolicies
	assert.Check(t, is.DeepEqual(values, expected))
}

func TestCompleteSignals(t *testing.T) {
	values, directives := completeSignals(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, len(values) > 1)
	assert.Check(t, is.Len(values, len(signal.SignalMap)))
}
