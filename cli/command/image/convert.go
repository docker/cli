package image

import (
	"context"
	"errors"
	"fmt"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/filters"
	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/morikuni/aec"
	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

type convertArgs struct {
	Src            string
	Dst            []string
	Platforms      []string
	NoAttestations bool
	OnlyAvailable  bool
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
	flags.BoolVar(&args.OnlyAvailable, "available", false, "Only include manifests which blobs are available locally")
	flags.StringArrayVar(&args.Dst, "to", nil, "Target image references")
	flags.StringVar(&args.Src, "from", "", "Source image reference")

	return cmd
}

type convertFilter = func(mfst imagetypes.ImageManifestSummary) bool

func runConvert(ctx context.Context, dockerCLI command.Cli, args convertArgs) error {
	if len(args.Dst) == 0 {
		return errors.New("no destination image specified")
	}
	if args.Src == "" {
		return errors.New("no source image specified")
	}

	matchesFilters, err := parseConvertFilters(args)
	if err != nil {
		return err
	}

	list, err := dockerCLI.Client().ImageList(ctx, imagetypes.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("reference", args.Src)),
	})
	if err != nil {
		return err
	}

	if len(list) == 0 {
		return fmt.Errorf("no such image: %s", args.Src)
	}

	newManifests := make([]imagetypes.ImageManifestSummary, 0, len(list[0].Manifests))
	for _, mfst := range list[0].Manifests {
		if !matchesFilters(mfst) {
			continue
		}
		newManifests = append(newManifests, mfst)
	}

	dstRefs := make([]reference.NamedTagged, 0, len(args.Dst))
	for _, dst := range args.Dst {
		dstRef, err := reference.ParseNormalizedNamed(dst)
		if err != nil {
			return fmt.Errorf("invalid destination image reference: %s: %w", dst, err)
		}

		dstRef = reference.TagNameOnly(dstRef)
		dstRefTagged := dstRef.(reference.NamedTagged)

		dstRefs = append(dstRefs, dstRefTagged)
	}

	newIndex := createIndex(newManifests)

	desc, err := dockerCLI.Client().ImageCreateFromOCIIndex(ctx, dstRefs[0], newIndex)
	if err != nil {
		return err
	}

	fmt.Println(aec.Bold.Apply("New image digest:"), desc.Digest.String())
	for idx, dst := range dstRefs {
		ref := reference.FamiliarString(dst)
		if idx > 0 {
			err := dockerCLI.Client().ImageTag(ctx, dstRefs[0].String(), dst.String())
			if err != nil {
				fmt.Print(aec.LightRedF.Apply(" ✗ "), ref+" - "+aec.LightRedF.Apply(" tag failed: "+err.Error()))
				continue
			}
		}

		fmt.Println(aec.LightGreenF.Apply(" ✓ "), ref)
	}

	return nil
}

func createIndex(manifests []imagetypes.ImageManifestSummary) v1.Index {
	idx := v1.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType: v1.MediaTypeImageIndex,
	}

	for _, mfst := range manifests {
		idx.Manifests = append(idx.Manifests, mfst.Descriptor)
	}
	return idx
}

func parseConvertFilters(args convertArgs) (convertFilter, error) {
	var flts []convertFilter

	// args.Platforms
	if len(args.Platforms) > 0 {
		f, err := filterPlatforms(args.Platforms)
		if err != nil {
			return nil, err
		}
		flts = append(flts, f)
	}

	// args.NoAttestations
	if args.NoAttestations {
		flts = append(flts, func(mfst imagetypes.ImageManifestSummary) bool {
			return mfst.Kind != imagetypes.ImageManifestKindAttestation
		})
	}

	// args.OnlyAvailablePlatforms
	if args.OnlyAvailable {
		flts = append(flts, func(mfst imagetypes.ImageManifestSummary) bool {
			return mfst.Available
		})
	}

	matchesFilters := func(mfst imagetypes.ImageManifestSummary) bool {
		for _, f := range flts {
			if !f(mfst) {
				return false
			}
		}
		return true
	}
	return matchesFilters, nil
}

func filterPlatforms(platformStrs []string) (convertFilter, error) {
	p := make([]v1.Platform, 0, len(platformStrs))
	for _, platform := range platformStrs {
		pl, err := platforms.Parse(platform)
		if err != nil {
			return nil, err
		}
		p = append(p, pl)
	}
	pm := platforms.Any(p...)

	return func(mfst imagetypes.ImageManifestSummary) bool {
		if mfst.Descriptor.Platform != nil {
			return pm.Match(*mfst.Descriptor.Platform)
		}

		if mfst.Kind != imagetypes.ImageManifestKindImage {
			return false
		}

		return pm.Match(mfst.ImageData.Platform)
	}, nil
}
