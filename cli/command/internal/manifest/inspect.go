package manifest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/distribution/reference"
	"github.com/docker/cli/cli/command/internal/cli"
	"github.com/docker/cli/cli/manifest/types"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	ref      string
	list     string
	verbose  bool
	insecure bool
}

// NewInspectCommand creates a new `docker manifest inspect` command
func newInspectCommand(dockerCLI cli.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] [MANIFEST_LIST] MANIFEST",
		Short: "Display an image manifest, or manifest list",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch len(args) {
			case 1:
				opts.ref = args[0]
			case 2:
				opts.list = args[0]
				opts.ref = args[1]
			}
			return runInspect(cmd.Context(), dockerCLI, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.insecure, "insecure", false, "Allow communication with an insecure registry")
	flags.BoolVarP(&opts.verbose, "verbose", "v", false, "Output additional info including layers and platform")
	return cmd
}

func runInspect(ctx context.Context, dockerCLI cli.Cli, opts inspectOptions) error {
	namedRef, err := normalizeReference(opts.ref)
	if err != nil {
		return err
	}

	// If list reference is provided, display the local manifest in a list
	if opts.list != "" {
		listRef, err := normalizeReference(opts.list)
		if err != nil {
			return err
		}

		imageManifest, err := newManifestStore(dockerCLI).Get(listRef, namedRef)
		if err != nil {
			return err
		}
		return printManifest(dockerCLI, imageManifest, opts)
	}

	// Try a local manifest list first
	localManifestList, err := newManifestStore(dockerCLI).GetList(namedRef)
	if err == nil {
		return printManifestList(dockerCLI, namedRef, localManifestList, opts)
	}

	// Next try a remote manifest
	registryClient := newRegistryClient(dockerCLI, opts.insecure)
	imageManifest, err := registryClient.GetManifest(ctx, namedRef)
	if err == nil {
		return printManifest(dockerCLI, imageManifest, opts)
	}

	// Finally try a remote manifest list
	manifestList, err := registryClient.GetManifestList(ctx, namedRef)
	if err != nil {
		return err
	}
	return printManifestList(dockerCLI, namedRef, manifestList, opts)
}

func printManifest(dockerCLI cli.Cli, manifest types.ImageManifest, opts inspectOptions) error {
	buffer := new(bytes.Buffer)
	if !opts.verbose {
		_, raw, err := manifest.Payload()
		if err != nil {
			return err
		}
		if err := json.Indent(buffer, raw, "", "\t"); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), buffer.String())
		return nil
	}
	jsonBytes, err := json.MarshalIndent(manifest, "", "\t")
	if err != nil {
		return err
	}
	_, _ = dockerCLI.Out().Write(append(jsonBytes, '\n'))
	return nil
}

func printManifestList(dockerCLI cli.Cli, namedRef reference.Named, list []types.ImageManifest, opts inspectOptions) error {
	if !opts.verbose {
		targetRepo := reference.TrimNamed(namedRef)

		manifests := make([]manifestlist.ManifestDescriptor, 0, len(list))
		// More than one response. This is a manifest list.
		for _, img := range list {
			mfd, err := buildManifestDescriptor(targetRepo, img)
			if err != nil {
				return fmt.Errorf("failed to assemble ManifestDescriptor: %w", err)
			}
			manifests = append(manifests, mfd)
		}
		deserializedML, err := manifestlist.FromDescriptors(manifests)
		if err != nil {
			return err
		}
		jsonBytes, err := deserializedML.MarshalJSON()
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), string(jsonBytes))
		return nil
	}
	jsonBytes, err := json.MarshalIndent(list, "", "\t")
	if err != nil {
		return err
	}
	_, _ = dockerCLI.Out().Write(append(jsonBytes, '\n'))
	return nil
}
