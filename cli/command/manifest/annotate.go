package manifest

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/manifest/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type annotateOptions struct {
	target     string // the target manifest list name (also transaction ID)
	image      string // the manifest to annotate within the list
	variant    string // an architecture variant
	os         string
	arch       string
	osFeatures []string
}

// NewAnnotateCommand creates a new `docker manifest annotate` command
func newAnnotateCommand(dockerCli command.Cli) *cobra.Command {
	var opts annotateOptions

	cmd := &cobra.Command{
		Use:   "annotate [OPTIONS] MANIFEST_LIST MANIFEST",
		Short: "Add additional information to a local image manifest",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.target = args[0]
			opts.image = args[1]
			return runManifestAnnotate(dockerCli, opts)
		},
	}

	flags := cmd.Flags()

	flags.StringVar(&opts.os, "os", "", "Set operating system")
	flags.StringVar(&opts.arch, "arch", "", "Set architecture")
	flags.StringSliceVar(&opts.osFeatures, "os-features", []string{}, "Set operating system feature")
	flags.StringVar(&opts.variant, "variant", "", "Set architecture variant")

	return cmd
}

func runManifestAnnotate(dockerCli command.Cli, opts annotateOptions) error {
	targetRef, err := normalizeReference(opts.target)
	if err != nil {
		return errors.Wrapf(err, "annotate: error parsing name for manifest list %s", opts.target)
	}
	imgRef, err := normalizeReference(opts.image)
	if err != nil {
		return errors.Wrapf(err, "annotate: error parsing name for manifest %s", opts.image)
	}

	manifestStore := dockerCli.ManifestStore()
	imageManifest, err := manifestStore.Get(targetRef, imgRef)
	switch {
	case store.IsNotFound(err):
		return fmt.Errorf("manifest for image %s does not exist in %s", opts.image, opts.target)
	case err != nil:
		return err
	}

	// Update the mf
	if opts.os != "" {
		imageManifest.Platform.OS = opts.os
	}
	if opts.arch != "" {
		imageManifest.Platform.Architecture = opts.arch
	}
	for _, osFeature := range opts.osFeatures {
		imageManifest.Platform.OSFeatures = appendIfUnique(imageManifest.Platform.OSFeatures, osFeature)
	}
	if opts.variant != "" {
		imageManifest.Platform.Variant = opts.variant
	}

	if !isValidOSArch(imageManifest.Platform.OS, imageManifest.Platform.Architecture) {
		return errors.Errorf("manifest entry for image has unsupported os/arch combination: %s/%s", opts.os, opts.arch)
	}
	return manifestStore.Save(targetRef, imgRef, imageManifest)
}

func appendIfUnique(list []string, str string) []string {
	for _, s := range list {
		if s == str {
			return list
		}
	}
	return append(list, str)
}
