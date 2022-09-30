package image

import (
	"context"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/sys/sequential"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type loadOptions struct {
	input string
	quiet bool
}

// NewLoadCommand creates a new `docker load` command
func NewLoadCommand(dockerCli command.Cli) *cobra.Command {
	var opts loadOptions

	cmd := &cobra.Command{
		Use:   "load [OPTIONS]",
		Short: "Load an image from a tar archive or STDIN",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoad(dockerCli, opts)
		},
		Annotations: map[string]string{
			"aliases": "docker image load, docker load",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.input, "input", "i", "", "Read from tar archive file, instead of STDIN")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress the load output")

	return cmd
}

func runLoad(dockerCli command.Cli, opts loadOptions) error {
	var input io.Reader = dockerCli.In()
	if opts.input != "" {
		// We use sequential.Open to use sequential file access on Windows, avoiding
		// depleting the standby list un-necessarily. On Linux, this equates to a regular os.Open.
		file, err := sequential.Open(opts.input)
		if err != nil {
			return err
		}
		defer file.Close()
		input = file
	}

	// To avoid getting stuck, verify that a tar file is given either in
	// the input flag or through stdin and if not display an error message and exit.
	if opts.input == "" && dockerCli.In().IsTerminal() {
		return errors.Errorf("requested load from stdin, but stdin is empty")
	}

	if !dockerCli.Out().IsTerminal() {
		opts.quiet = true
	}
	response, err := dockerCli.Client().ImageLoad(context.Background(), input, opts.quiet)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.Body != nil && response.JSON {
		return jsonmessage.DisplayJSONMessagesToStream(response.Body, dockerCli.Out(), nil)
	}

	_, err = io.Copy(dockerCli.Out(), response.Body)
	return err
}
