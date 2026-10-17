// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.26

package manifest

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/manifest/store"
	"github.com/docker/cli/internal/hint"
	"github.com/docker/cli/internal/registryclient"
	"github.com/moby/moby/api/types/registry"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
	cfg := dockerCLI.ConfigFile()
	resolver := func(ctx context.Context, domainName string) registry.AuthConfig {
		a, _ := cfg.GetAuthConfig(domainName)
		return registry.AuthConfig{
			Username:      a.Username,
			Password:      a.Password,
			ServerAddress: a.ServerAddress,

			// TODO(thaJeztah): Are these expected to be included?
			Auth:          a.Auth,
			IdentityToken: a.IdentityToken,
			RegistryToken: a.RegistryToken,
		}
	}
	// FIXME(thaJeztah): this should use the userAgent as configured on the dockerCLI.
	return registryclient.NewRegistryClient(resolver, command.UserAgent(), allowInsecure)
}

// NewAnnotateCommand creates a new `docker manifest annotate` command
func newAnnotateCommand(dockerCLI command.Cli) *cobra.Command {
	var opts annotateOptions

	cmd := &cobra.Command{
		Use:   "annotate [OPTIONS] MANIFEST_LIST MANIFEST",
		Short: "Add additional information to a local image manifest",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.target = args[0]
			opts.image = args[1]
			return runManifestAnnotate(dockerCLI, opts)
		},
		DisableFlagsInUseLine: true,
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
		return fmt.Errorf("annotate: error parsing name for manifest list %s: %w", opts.target, err)
	}
	imgRef, err := normalizeReference(opts.image)
	if err != nil {
		return fmt.Errorf("annotate: error parsing name for manifest %s: %w", opts.image, err)
	}

	manifestStore := newManifestStore(dockerCLI)
	imageManifest, err := manifestStore.Get(targetRef, imgRef)
	switch {
	case errdefs.IsNotFound(err):
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
		return hint.Wrap(
			fmt.Errorf("unsupported os/arch combination %s/%s for manifest entry %s", imageManifest.Descriptor.Platform.OS, imageManifest.Descriptor.Platform.Architecture, opts.image),
			"Set --os and --arch to a supported pair (for example linux/amd64, linux/arm64, or windows/amd64).",
		)
	}
	return manifestStore.Save(targetRef, imgRef, imageManifest)
}

func appendIfUnique(list []string, str string) []string {
	if slices.Contains(list, str) {
		return list
	}
	return append(list, str)
}
