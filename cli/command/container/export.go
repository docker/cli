package container

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/internal/hint"
	"github.com/moby/moby/client"
	"github.com/moby/sys/atomicwriter"
	"github.com/spf13/cobra"
)

type exportOptions struct {
	container string
	output    string
}

// newExportCommand creates a new "docker container export" command.
func newExportCommand(dockerCLI command.Cli) *cobra.Command {
	var opts exportOptions

	cmd := &cobra.Command{
		Use:   "export [OPTIONS] CONTAINER",
		Short: "Export a container's filesystem as a tar archive",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.container = args[0]
			return runExport(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container export, docker export",
		},
		ValidArgsFunction:     completion.ContainerNames(dockerCLI, true),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.output, "output", "o", "", "Write to a file, instead of STDOUT")

	return cmd
}

func runExport(ctx context.Context, dockerCLI command.Cli, opts exportOptions) error {
	var output io.Writer
	if opts.output == "" {
		if dockerCLI.Out().IsTerminal() {
			return hint.Wrap(
				errors.New("refusing to write a binary tar archive to the terminal"),
				"Use '-o FILE' to write to a file, or redirect stdout, e.g. 'docker container export CONTAINER > out.tar'.",
			)
		}
		output = dockerCLI.Out()
	} else {
		writer, err := atomicwriter.New(opts.output, 0o600)
		if err != nil {
			return fmt.Errorf("cannot open output file %q: %w", opts.output, err)
		}
		defer writer.Close()
		output = writer
	}

	responseBody, err := dockerCLI.Client().ContainerExport(ctx, opts.container, client.ContainerExportOptions{})
	if err != nil {
		return err
	}
	defer responseBody.Close()

	_, err = io.Copy(output, responseBody)
	return err
}
