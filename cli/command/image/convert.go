package image

import (
	"context"
	"fmt"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/spf13/cobra"
)

type convertArgs struct {
	Src                    string
	Dst                    []string
	Platforms              []string
	NoAttestations         bool
	OnlyAvailablePlatforms bool
}

func NewConvertCommand(dockerCli command.Cli) *cobra.Command {
	var args convertArgs

	cmd := &cobra.Command{
		Use:   "convert [OPTIONS]",
		Short: "Convert multi-platform images",
		Args:  cli.ExactArgs(0),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConvert(cmd.Context(), dockerCli, args)
		},
		Aliases: []string{"convert"},
		Annotations: map[string]string{
			"aliases": "docker image convert, docker convert",
		},
	}

	flags := cmd.Flags()
	flags.StringArrayVar(&args.Platforms, "platforms", nil, "Include only the specified platforms in the destination image")
	flags.BoolVar(&args.NoAttestations, "no-attestations", false, "Do not include image attestations")
	flags.BoolVar(&args.OnlyAvailablePlatforms, "available", false, "Only include platforms locally available to the daemon")
	flags.StringArrayVar(&args.Dst, "to", nil, "Target image references")
	flags.StringVar(&args.Src, "from", "", "Source image reference")

	return cmd
}

func runConvert(ctx context.Context, dockerCLI command.Cli, args convertArgs) error {
	if len(args.Dst) == 0 {
		return fmt.Errorf("No destination image specified")
	}
	if args.Src == "" {
		return fmt.Errorf("No source image specified")
	}

	var dstRefs []reference.NamedTagged
	for _, dst := range args.Dst {
		dstRef, err := reference.ParseNormalizedNamed(dst)
		if err != nil {
			return fmt.Errorf("invalid destination image reference: %s: %w", dst, err)
		}

		dstRef = reference.TagNameOnly(dstRef)
		dstRefTagged := dstRef.(reference.NamedTagged)
		dstRefs = append(dstRefs, dstRefTagged)
	}

	opts := imagetypes.ConvertOptions{
		NoAttestations:         args.NoAttestations,
		OnlyAvailablePlatforms: args.OnlyAvailablePlatforms,
	}

	for _, platform := range args.Platforms {
		p, err := platforms.Parse(platform)
		if err != nil {
			return err
		}
		opts.Platforms = append(opts.Platforms, p)
	}

	return dockerCLI.Client().ImageConvert(ctx, args.Src, dstRefs, opts)
}
