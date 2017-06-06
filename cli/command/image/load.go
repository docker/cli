package image

import (
	"io"

	"golang.org/x/net/context"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/system"
	runconfigopts "github.com/docker/docker/runconfig/opts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type loadOptions struct {
	input string
	quiet bool
	name  string
	refs  []string
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
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.input, "input", "i", "", "Read from tar archive file, instead of STDIN")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress the load output")
	flags.StringVarP(&opts.name, "name", "n", "", "Name to use when loading OCI image layout tar archive")
	flags.StringSliceVar(&opts.refs, "ref", []string{}, "References to use when loading an OCI image layout tar archive")

	return cmd
}

func runLoad(dockerCli command.Cli, opts loadOptions) error {
	var input io.Reader = dockerCli.In()
	if opts.input != "" {
		// We use system.OpenSequential to use sequential file access on Windows, avoiding
		// depleting the standby list un-necessarily. On Linux, this equates to a regular os.Open.
		file, err := system.OpenSequential(opts.input)
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
	imageLoadOpts := types.ImageLoadOptions{
		Quiet: opts.quiet,
		Name:  opts.name,
		Refs:  runconfigopts.ConvertKVStringsToMap(opts.refs),
	}
	response, err := dockerCli.Client().ImageLoad(context.Background(), input, imageLoadOpts)
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
