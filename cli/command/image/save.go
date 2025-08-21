package image

import (
	"context"
	"io"

	"github.com/containerd/platforms"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/moby/moby/client"
	"github.com/moby/sys/atomicwriter"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type saveOptions struct {
	images   []string
	output   string
	platform []string
}

// newSaveCommand creates a new "docker image save" command.
func newSaveCommand(dockerCLI command.Cli) *cobra.Command {
	var opts saveOptions

	cmd := &cobra.Command{
		Use:   "save [OPTIONS] IMAGE [IMAGE...]",
		Short: "Save one or more images to a tar archive (streamed to STDOUT by default)",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.images = args
			return runSave(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"aliases": "docker image save, docker save",
		},
		ValidArgsFunction: completion.ImageNames(dockerCLI, -1),
	}

	flags := cmd.Flags()

	flags.StringVarP(&opts.output, "output", "o", "", "Write to a file, instead of STDOUT")
	flags.StringSliceVar(&opts.platform, "platform", []string{}, `Save only the given platform(s). Formatted as a comma-separated list of "os[/arch[/variant]]" (e.g., "linux/amd64,linux/arm64/v8")`)
	_ = flags.SetAnnotation("platform", "version", []string{"1.48"})

	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)
	return cmd
}

// runSave performs a save against the engine based on the specified options
func runSave(ctx context.Context, dockerCLI command.Cli, opts saveOptions) error {
	var options []client.ImageSaveOption

	platformList := []ocispec.Platform{}
	for _, p := range opts.platform {
		pp, err := platforms.Parse(p)
		if err != nil {
			return errors.Wrap(err, "invalid platform")
		}
		platformList = append(platformList, pp)
	}
	if len(platformList) > 0 {
		options = append(options, client.ImageSaveWithPlatforms(platformList...))
	}

	var output io.Writer
	if opts.output == "" {
		if dockerCLI.Out().IsTerminal() {
			return errors.New("cowardly refusing to save to a terminal. Use the -o flag or redirect")
		}
		output = dockerCLI.Out()
	} else {
		writer, err := atomicwriter.New(opts.output, 0o600)
		if err != nil {
			return errors.Wrap(err, "failed to save image")
		}
		defer writer.Close()
		output = writer
	}

	responseBody, err := dockerCLI.Client().ImageSave(ctx, opts.images, options...)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	_, err = io.Copy(output, responseBody)
	return err
}
