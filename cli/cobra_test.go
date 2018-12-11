package cli

import (
	"testing"

	pluginmanager "github.com/docker/cli/cli-plugins/manager"
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

func TestCommandVendor(t *testing.T) {
	// Non plugin.
	assert.Equal(t, commandVendor(&cobra.Command{Use: "test"}), "             ")

	// Plugins with various lengths of vendor.
	for _, tc := range []struct {
		vendor   string
		expected string
	}{
		{vendor: "vendor", expected: "(vendor)     "},
		{vendor: "vendor12345", expected: "(vendor12345)"},
		{vendor: "vendor123456", expected: "(vendor1234…)"},
		{vendor: "vendor1234567", expected: "(vendor1234…)"},
	} {
		t.Run(tc.vendor, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "test",
				Annotations: map[string]string{
					pluginmanager.CommandAnnotationPluginVendor: tc.vendor,
				},
			}
			assert.Equal(t, commandVendor(cmd), tc.expected)
		})
	}
}
