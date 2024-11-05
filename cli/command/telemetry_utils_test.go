package command

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/cli/cli/streams"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
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
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			actual := getCommandName(tc.cmd)
			assert.Equal(t, actual, tc.expected)
		})
	}
}

func TestStdioAttributes(t *testing.T) {
	outBuffer := new(bytes.Buffer)
	errBuffer := new(bytes.Buffer)
	t.Parallel()
	for _, tc := range []struct {
		test      string
		stdinTty  bool
		stdoutTty bool
		// TODO(laurazard): test stderr
		expected []attribute.KeyValue
	}{
		{
			test: "",
			expected: []attribute.KeyValue{
				attribute.Bool("command.stdin.isatty", false),
				attribute.Bool("command.stdout.isatty", false),
				attribute.Bool("command.stderr.isatty", false),
			},
		},
		{
			test:      "",
			stdinTty:  true,
			stdoutTty: true,
			expected: []attribute.KeyValue{
				attribute.Bool("command.stdin.isatty", true),
				attribute.Bool("command.stdout.isatty", true),
				attribute.Bool("command.stderr.isatty", false),
			},
		},
	} {
		t.Run(tc.test, func(t *testing.T) {
			t.Parallel()
			cli := &DockerCli{
				in:  streams.NewIn(io.NopCloser(strings.NewReader(""))),
				out: streams.NewOut(outBuffer),
				err: streams.NewOut(errBuffer),
			}
			cli.In().SetIsTerminal(tc.stdinTty)
			cli.Out().SetIsTerminal(tc.stdoutTty)
			actual := stdioAttributes(cli)

			assert.Check(t, reflect.DeepEqual(actual, tc.expected))
		})
	}
}

func TestAttributesFromError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		testName string
		err      error
		expected []attribute.KeyValue
	}{
		{
			testName: "no error",
			err:      nil,
			expected: []attribute.KeyValue{
				attribute.Int("command.status.code", 0),
			},
		},
		{
			testName: "non-0 exit code",
			err:      statusError{StatusCode: 127},
			expected: []attribute.KeyValue{
				attribute.String("command.error.type", "generic"),
				attribute.Int("command.status.code", 127),
			},
		},
		{
			testName: "canceled",
			err:      context.Canceled,
			expected: []attribute.KeyValue{
				attribute.String("command.error.type", "canceled"),
				attribute.Int("command.status.code", 1),
			},
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			t.Parallel()
			actual := attributesFromError(tc.err)
			assert.Check(t, reflect.DeepEqual(actual, tc.expected))
		})
	}
}
