package plugin

import (
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type loadOpts struct {
	input string
	quiet bool
}

func newLoadCommand(dockerCli command.Cli) *cobra.Command {
	var opts loadOpts

	cmd := &cobra.Command{
		Use:   "load [OPTIONS]",
		Short: "Load a plugin from a tar archive or STDIN",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoad(dockerCli, &opts)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.input, "input", "i", "", "Read from tar archive file, instead of STDIN")
	// TODO: load needs a --grant-all-permissions flag. The implementation should be similar to
	// acceptPrivileges, the main difference being no involvement with registry and hence
	// lack of registryAuth.
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress the load output")

	return cmd
}

func runLoad(dockerCli command.Cli, opts *loadOpts) error {
	var input io.Reader = dockerCli.In()
	if opts.input != "" {
		file, err := os.Open(opts.input)
		if err != nil {
			return err
		}
		defer file.Close()
		input = file
	}

	if opts.input == "" && dockerCli.In().IsTerminal() {
		return errors.Errorf("requested load from stdin, but stdin is empty")
	}

	if !dockerCli.Out().IsTerminal() {
		opts.quiet = true
	}

	// response is the progress
	response, err := dockerCli.Client().PluginLoad(context.Background(), input, opts.quiet)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.Body != nil && response.JSON {
		return jsonmessage.DisplayJSONMessagesToStream(response.Body, dockerCli.Out(), nil)
	}

	// copy the response to the docker cli output
	_, err = io.Copy(dockerCli.Out(), response.Body)
	return err
}
