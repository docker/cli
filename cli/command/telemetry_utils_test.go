package command

import (
	"testing"

	"github.com/spf13/cobra"
	"gotest.tools/v3/assert"
)

func setupCobraCommands() (*cobra.Command, *cobra.Command, *cobra.Command) {
	rootCmd := &cobra.Command{
		Use: "root [OPTIONS] COMMAND [ARG...]",
	}
	childCmd := &cobra.Command{
		Use: "child [OPTIONS] COMMAND [ARG...]",
	}
	grandchildCmd := &cobra.Command{
		Use: "grandchild [OPTIONS] COMMAND [ARG...]",
	}
	childCmd.AddCommand(grandchildCmd)
	rootCmd.AddCommand(childCmd)

	return rootCmd, childCmd, grandchildCmd
}

func TestGetFullCommandName(t *testing.T) {
	rootCmd, childCmd, grandchildCmd := setupCobraCommands()

	t.Parallel()

	for _, tc := range []struct {
		testName string
		cmd      *cobra.Command
		expected string
	}{
		{
			testName: "rootCmd",
			cmd:      rootCmd,
			expected: "root",
		},
		{
			testName: "childCmd",
			cmd:      childCmd,
			expected: "root child",
		},
		{
			testName: "grandChild",
			cmd:      grandchildCmd,
			expected: "root child grandchild",
		},
	} {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			actual := getFullCommandName(tc.cmd)
			assert.Equal(t, actual, tc.expected)
		})
	}
}

func TestGetCommandName(t *testing.T) {
	rootCmd, childCmd, grandchildCmd := setupCobraCommands()

	t.Parallel()

	for _, tc := range []struct {
		testName string
		cmd      *cobra.Command
		expected string
	}{
		{
			testName: "rootCmd",
			cmd:      rootCmd,
			expected: "",
		},
		{
			testName: "childCmd",
			cmd:      childCmd,
			expected: "child",
		},
		{
			testName: "grandchildCmd",
			cmd:      grandchildCmd,
			expected: "child grandchild",
		},
	} {
		tc := tc
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			actual := getCommandName(tc.cmd)
			assert.Equal(t, actual, tc.expected)
		})
	}
}
