package stack

import (
	"context"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/inspect"
	apiclient "github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	refs   []string
	format string
	// pretty bool // TODO - add support for pretty rendering of the stack
}

func newInspectCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] STACK [STACK...]",
		Short: "Display detailed information on one or more stacks",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.refs = args
			return runInspect(dockerCli, opts, common.Orchestrator())
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")
	// flags.BoolVar(&opts.pretty, "pretty", false, "Print the information in a human friendly format") // TODO
	return cmd
}

func runInspect(dockerCli command.Cli, opts inspectOptions, commonOrchestrator command.Orchestrator) error {
	ctx := context.Background()

	// Stack inspect only supported on newer engines with server-side
	// stack support
	if !hasServerSideStacks(dockerCli) {
		return errors.Errorf("Error: your engine is too old for the inspect command")
	}

	getRef := func(ref string) (interface{}, []byte, error) {
		stack, err := getStackByName(ctx, dockerCli, string(commonOrchestrator), ref)
		if err == nil || !apiclient.IsErrNotFound(err) {
			return stack, nil, err
		}
		return nil, nil, errors.Errorf("Error: no such stack: %s", ref)
	}

	f := opts.format
	if len(f) == 0 {
		f = "raw"
	}

	// TODO - add pretty support
	err := inspect.Inspect(
		dockerCli.Out(),
		opts.refs,
		string(formatter.Format(strings.TrimPrefix(f, formatter.RawFormatKey))),
		getRef)

	if err != nil {
		return cli.StatusError{StatusCode: 1, Status: err.Error()}
	}
	return nil
}
