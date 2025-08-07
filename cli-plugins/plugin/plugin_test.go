package plugin

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"
)

func TestVisitAll(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	sub1 := &cobra.Command{Use: "sub1"}
	sub1sub1 := &cobra.Command{Use: "sub1sub1"}
	sub1sub2 := &cobra.Command{Use: "sub1sub2"}
	sub2 := &cobra.Command{Use: "sub2"}

	root.AddCommand(sub1, sub2)
	sub1.AddCommand(sub1sub1, sub1sub2)

	var visited []string
	visitAll(root, func(ccmd *cobra.Command) {
		visited = append(visited, ccmd.Name())
	})
	expected := []string{"sub1sub1", "sub1sub2", "sub1", "sub2", "root"}
	if !slices.Equal(expected, visited) {
		t.Errorf("expected %#v, got %#v", expected, visited)
	}
}
