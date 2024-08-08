package image

import (
	"context"
	"fmt"
	"io"

	"github.com/containerd/platforms"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/docker/api/types/image"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type saveOptions struct {
	images   []string
	output   string
	platform string
}

// NewSaveCommand creates a new `docker save` command
func NewSaveCommand(dockerCli command.Cli) *cobra.Command {
	var opts saveOptions

	cmd := &cobra.Command{
		Use:   "save [OPTIONS] IMAGE [IMAGE...]",
		Short: "Save one or more images to a tar archive (streamed to STDOUT by default)",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.images = args
			return RunSave(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"aliases": "docker image save, docker save",
		},
		ValidArgsFunction: completion.ImageNames(dockerCli),
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.output, "output", "o", "", "Write to a file, instead of STDOUT")
	flags.StringVar(&opts.platform, "platform", "",
		`Pick a single-platform to be saved if the image is multi-platform.
Full multi-platform image will be saved if not specified.

Format: os[/arch[/variant]]
Example: "linux/amd64"`)
	flags.SetAnnotation("platform", "version", []string{"1.47"})

	return cmd
}

// RunSave performs a save against the engine based on the specified options
func RunSave(ctx context.Context, dockerCli command.Cli, opts saveOptions) error {
	if opts.output == "" && dockerCli.Out().IsTerminal() {
		return errors.New("cowardly refusing to save to a terminal. Use the -o flag or redirect")
	}

	if err := command.ValidateOutputPath(opts.output); err != nil {
		return errors.Wrap(err, "failed to save image")
	}

	var saveOptions image.SaveOptions
	if opts.platform != "" {
		p, err := platforms.Parse(opts.platform)
		if err != nil {
			_, _ = fmt.Fprintf(dockerCli.Err(), "Invalid platform %s", opts.platform)
			return err
		}
		saveOptions.Platform = &p
	}

	responseBody, err := dockerCli.Client().ImageSave(ctx, opts.images, saveOptions)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	if opts.output == "" {
		_, err := io.Copy(dockerCli.Out(), responseBody)
		return err
	}

	return command.CopyToFile(opts.output, responseBody)
}
