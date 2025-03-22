package container

import (
	"context"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/pkg/atomicwriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type exportOptions struct {
	container string
	output    string
}

// NewExportCommand creates a new `docker export` command
func NewExportCommand(dockerCli command.Cli) *cobra.Command {
	var opts exportOptions

	cmd := &cobra.Command{
		Use:   "export [OPTIONS] CONTAINER",
		Short: "Export a container's filesystem as a tar archive",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.container = args[0]
			return runExport(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"aliases": "docker container export, docker export",
		},
		ValidArgsFunction: completion.ContainerNames(dockerCli, true),
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.output, "output", "o", "", "Write to a file, instead of STDOUT")

	return cmd
}

func runExport(ctx context.Context, dockerCli command.Cli, opts exportOptions) error {
	var output io.Writer
	if opts.output == "" {
		if dockerCli.Out().IsTerminal() {
			return errors.New("cowardly refusing to save to a terminal. Use the -o flag or redirect")
		}
		output = dockerCli.Out()
	} else {
		if err := command.ValidateOutputPath(opts.output); err != nil {
			return errors.Wrap(err, "failed to export container")
		}
		writer, err := atomicwriter.New(opts.output, 0o600)
		if err != nil {
			return err
		}
		defer writer.Close()
		output = writer
	}

	responseBody, err := dockerCli.Client().ContainerExport(ctx, opts.container)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	_, err = io.Copy(output, responseBody)
	return err
}
