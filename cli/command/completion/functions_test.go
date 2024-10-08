package completion

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestCompletePlatforms(t *testing.T) {
	values, directives := Platforms(nil, nil, "")
	assert.Check(t, is.Equal(directives&cobra.ShellCompDirectiveNoFileComp, cobra.ShellCompDirectiveNoFileComp), "Should not perform file completion")
	assert.Check(t, is.DeepEqual(values, commonPlatforms))
}
