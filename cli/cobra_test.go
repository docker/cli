package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/assert"
)

func TestVisitAll(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	sub1 := &cobra.Command{Use: "sub1"}
	sub1sub1 := &cobra.Command{Use: "sub1sub1"}
	sub1sub2 := &cobra.Command{Use: "sub1sub2"}
	sub2 := &cobra.Command{Use: "sub2"}

	root.AddCommand(sub1, sub2)
	sub1.AddCommand(sub1sub1, sub1sub2)

	// Take the opportunity to test DisableFlagsInUseLine too
	DisableFlagsInUseLine(root)

	var visited []string
	VisitAll(root, func(ccmd *cobra.Command) {
		visited = append(visited, ccmd.Name())
		assert.Assert(t, ccmd.DisableFlagsInUseLine, "DisableFlagsInUseLine not set on %q", ccmd.Name())
	})
	expected := []string{"sub1sub1", "sub1sub2", "sub1", "sub2", "root"}
	assert.DeepEqual(t, expected, visited)
}
