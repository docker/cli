package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/jsonstream"
	"github.com/docker/cli/templates"
	"github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

// pullOptions defines what and how to pull.
type pullOptions struct {
	remote   string
	all      bool
	platform string
	quiet    bool
	dryRun   bool
	format   string
}

// newPullCommand creates a new `docker pull` command
func newPullCommand(dockerCLI command.Cli) *cobra.Command {
	var opts pullOptions

	cmd := &cobra.Command{
		Use:   "pull [OPTIONS] NAME[:TAG|@DIGEST]",
		Short: "Download an image from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			if opts.format != "" && !opts.dryRun {
				return errors.New("--format can only be used with --dry-run")
			}
			return runPull(cmd.Context(), dockerCLI, opts)
		},
		Annotations: map[string]string{
			"category-top": "5",
			"aliases":      "docker image pull, docker pull",
		},
		// Complete with local images to help pulling the latest version
		// of images that are in the image cache.
		ValidArgsFunction:     completion.ImageNames(dockerCLI, 1),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()

	flags.BoolVarP(&opts.all, "all-tags", "a", false, "Download all tagged images in the repository")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "Check if a newer image is available without pulling")
	flags.StringVar(&opts.format, "format", "", "Format output of --dry-run using a custom template: 'json' or 'TEMPLATE'")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress verbose output")
	cmd.MarkFlagsMutuallyExclusive("all-tags", "dry-run")

	// TODO(thaJeztah): DEPRECATED: remove in v29.1 or v30
	flags.Bool("disable-content-trust", true, "Skip image verification (deprecated)")
	_ = flags.MarkDeprecated("disable-content-trust", "support for docker content trust was removed")

	flags.StringVar(&opts.platform, "platform", os.Getenv("DOCKER_DEFAULT_PLATFORM"), "Set platform if server is multi-platform capable")
	_ = flags.SetAnnotation("platform", "version", []string{"1.32"})
	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms())

	return cmd
}

// runPull performs a pull against the engine based on the specified options
func runPull(ctx context.Context, dockerCLI command.Cli, opts pullOptions) error {
	distributionRef, err := reference.ParseNormalizedNamed(opts.remote)
	switch {
	case err != nil:
		return err
	case opts.all && !reference.IsNameOnly(distributionRef):
		return errors.New("tag can't be used with --all-tags/-a")
	case !opts.all && reference.IsNameOnly(distributionRef):
		distributionRef = reference.TagNameOnly(distributionRef)
		if tagged, ok := distributionRef.(reference.Tagged); ok && !opts.quiet {
			_, _ = fmt.Fprintln(dockerCLI.Out(), "Using default tag:", tagged.Tag())
		}
	}

	var ociPlatforms []ocispec.Platform
	if opts.platform != "" {
		// TODO(thaJeztah): add a platform option-type / flag-type.
		p, err := platforms.Parse(opts.platform)
		if err != nil {
			return err
		}
		ociPlatforms = append(ociPlatforms, p)
	}

	encodedAuth, err := command.RetrieveAuthTokenFromImage(dockerCLI.ConfigFile(), distributionRef.String())
	if err != nil {
		return err
	}

	if opts.dryRun {
		return runPullDryRun(ctx, dockerCLI, distributionRef, encodedAuth, opts)
	}

	responseBody, err := dockerCLI.Client().ImagePull(ctx, reference.FamiliarString(distributionRef), client.ImagePullOptions{
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: nil,
		All:           opts.all,
		Platforms:     ociPlatforms,
	})
	if err != nil {
		return err
	}
	defer responseBody.Close()

	out := dockerCLI.Out()
	if opts.quiet {
		out = streams.NewOut(io.Discard)
	}
	if err := jsonstream.Display(ctx, responseBody, out); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(dockerCLI.Out(), distributionRef.String())
	return nil
}

// Status values reported by "docker pull --dry-run".
const (
	dryRunUpToDate       = "up-to-date"
	dryRunNewerAvailable = "newer-available"
	dryRunNotPresent     = "not-present"
)

// dryRunResult is the result of "docker pull --dry-run". Its fields can be
// used in the --format template.
type dryRunResult struct {
	Reference string
	Digest    string
	Status    string
}

// runPullDryRun checks whether pulling the reference would download a newer
// image, without pulling it. It asks the daemon to resolve the reference to
// a digest in the registry, and compares that digest against the digests of
// the local image.
func runPullDryRun(ctx context.Context, dockerCLI command.Cli, distributionRef reference.Named, encodedAuth string, opts pullOptions) error {
	// Validate the format before contacting the registry.
	var tmpl *template.Template
	if opts.format != "" && opts.format != "json" {
		var err error
		if tmpl, err = templates.Parse(opts.format); err != nil {
			return fmt.Errorf("template parsing error: %w", err)
		}
	}

	remote, err := dockerCLI.Client().DistributionInspect(ctx, reference.FamiliarString(distributionRef), client.DistributionInspectOptions{
		EncodedRegistryAuth: encodedAuth,
	})
	if err != nil {
		return err
	}
	remoteDigest := remote.Descriptor.Digest
	if remoteDigest == "" {
		return fmt.Errorf("registry did not return a digest for %s", distributionRef)
	}

	result := dryRunResult{Reference: distributionRef.String(), Digest: remoteDigest.String()}
	local, err := dockerCLI.Client().ImageInspect(ctx, reference.FamiliarString(distributionRef))
	switch {
	case errdefs.IsNotFound(err):
		result.Status = dryRunNotPresent
	case err != nil:
		return err
	case slices.ContainsFunc(local.RepoDigests, func(repoDigest string) bool {
		return strings.HasSuffix(repoDigest, "@"+result.Digest)
	}):
		result.Status = dryRunUpToDate
	default:
		result.Status = dryRunNewerAvailable
	}

	switch {
	case opts.format == "json":
		b, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(dockerCLI.Out(), string(b))
	case tmpl != nil:
		if err := tmpl.Execute(dockerCLI.Out(), result); err != nil {
			return fmt.Errorf("template parsing error: %w", err)
		}
		_, _ = fmt.Fprintln(dockerCLI.Out())
	default:
		printDryRunText(dockerCLI, result, opts.quiet)
	}
	return nil
}

func printDryRunText(dockerCLI command.Cli, result dryRunResult, quiet bool) {
	out := dockerCLI.Out()
	if quiet {
		out = streams.NewOut(io.Discard)
	}
	_, _ = fmt.Fprintln(out, "Digest:", result.Digest)
	switch result.Status {
	case dryRunNotPresent:
		_, _ = fmt.Fprintf(out, "Status: Image %s is not available locally and would be pulled\n", result.Reference)
	case dryRunUpToDate:
		_, _ = fmt.Fprintf(out, "Status: Image is up to date for %s\n", result.Reference)
	default:
		_, _ = fmt.Fprintf(out, "Status: A newer image is available for %s\n", result.Reference)
	}
	_, _ = fmt.Fprintln(dockerCLI.Out(), result.Reference)
}
