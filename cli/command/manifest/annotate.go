package manifest

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/manifest/store"
	registryclient "github.com/docker/cli/cli/registry/client"
	"github.com/docker/docker/api/types/registry"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
	osVersion  string
}

// manifestStoreProvider is used in tests to provide a dummy store.
type manifestStoreProvider interface {
	// ManifestStore returns a store for local manifests
	ManifestStore() store.Store
	RegistryClient(bool) registryclient.RegistryClient
}

// newManifestStore returns a store for local manifests
func newManifestStore(dockerCLI command.Cli) store.Store {
	if msp, ok := dockerCLI.(manifestStoreProvider); ok {
		// manifestStoreProvider is used in tests to provide a dummy store.
		return msp.ManifestStore()
	}

	// TODO: support override default location from config file
	return store.NewStore(filepath.Join(config.Dir(), "manifests"))
}

// newRegistryClient returns a client for communicating with a Docker distribution
// registry
func newRegistryClient(dockerCLI command.Cli, allowInsecure bool) registryclient.RegistryClient {
	if msp, ok := dockerCLI.(manifestStoreProvider); ok {
		// manifestStoreProvider is used in tests to provide a dummy store.
		return msp.RegistryClient(allowInsecure)
	}
	resolver := func(ctx context.Context, index *registry.IndexInfo) registry.AuthConfig {
		return command.ResolveAuthConfig(dockerCLI.ConfigFile(), index)
	}
	return registryclient.NewRegistryClient(resolver, command.UserAgent(), allowInsecure)
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
	flags.StringVar(&opts.osVersion, "os-version", "", "Set operating system version")
	flags.StringSliceVar(&opts.osFeatures, "os-features", []string{}, "Set operating system feature")
	flags.StringVar(&opts.variant, "variant", "", "Set architecture variant")

	return cmd
}

func runManifestAnnotate(dockerCLI command.Cli, opts annotateOptions) error {
	targetRef, err := normalizeReference(opts.target)
	if err != nil {
		return errors.Wrapf(err, "annotate: error parsing name for manifest list %s", opts.target)
	}
	imgRef, err := normalizeReference(opts.image)
	if err != nil {
		return errors.Wrapf(err, "annotate: error parsing name for manifest %s", opts.image)
	}

	manifestStore := newManifestStore(dockerCLI)
	imageManifest, err := manifestStore.Get(targetRef, imgRef)
	switch {
	case store.IsNotFound(err):
		return fmt.Errorf("manifest for image %s does not exist in %s", opts.image, opts.target)
	case err != nil:
		return err
	}

	// Update the mf
	if imageManifest.Descriptor.Platform == nil {
		imageManifest.Descriptor.Platform = new(ocispec.Platform)
	}
	if opts.os != "" {
		imageManifest.Descriptor.Platform.OS = opts.os
	}
	if opts.arch != "" {
		imageManifest.Descriptor.Platform.Architecture = opts.arch
	}
	for _, osFeature := range opts.osFeatures {
		imageManifest.Descriptor.Platform.OSFeatures = appendIfUnique(imageManifest.Descriptor.Platform.OSFeatures, osFeature)
	}
	if opts.variant != "" {
		imageManifest.Descriptor.Platform.Variant = opts.variant
	}
	if opts.osVersion != "" {
		imageManifest.Descriptor.Platform.OSVersion = opts.osVersion
	}

	if !isValidOSArch(imageManifest.Descriptor.Platform.OS, imageManifest.Descriptor.Platform.Architecture) {
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
