// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.21

package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types/auxprogress"
	"github.com/docker/docker/api/types/image"
	imagetypes "github.com/docker/docker/api/types/image"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/registry"
	"github.com/morikuni/aec"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	all       bool
	remote    string
	untrusted bool
	quiet     bool
	platform  string
}

// NewPushCommand creates a new `docker push` command
func NewPushCommand(dockerCli command.Cli) *cobra.Command {
	var opts pushOptions

	cmd := &cobra.Command{
		Use:   "push [OPTIONS] NAME[:TAG]",
		Short: "Upload an image to a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.remote = args[0]
			return RunPush(cmd.Context(), dockerCli, opts)
		},
		Annotations: map[string]string{
			"category-top": "6",
			"aliases":      "docker image push, docker push",
		},
		ValidArgsFunction: completion.ImageNames(dockerCli),
	}

	flags := cmd.Flags()
	flags.BoolVarP(&opts.all, "all-tags", "a", false, "Push all tags of an image to the repository")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress verbose output")
	command.AddTrustSigningFlags(flags, &opts.untrusted, dockerCli.ContentTrustEnabled())

	// Don't default to DOCKER_DEFAULT_PLATFORM env variable, always default to
	// pushing the image as-is. This also avoids forcing the platform selection
	// on older APIs which don't support it.
	flags.StringVar(&opts.platform, "platform", "",
		`Push a platform-specific manifest as a single-platform image to the registry.
Image index won't be pushed, meaning that other manifests, including attestations won't be preserved.
'os[/arch[/variant]]': Explicit platform (eg. linux/amd64)`)
	flags.SetAnnotation("platform", "version", []string{"1.46"})

	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)

	return cmd
}

// RunPush performs a push against the engine based on the specified options
//
//nolint:gocyclo
func RunPush(ctx context.Context, dockerCli command.Cli, opts pushOptions) error {
	var platform *ocispec.Platform
	if opts.platform != "" {
		p, err := platforms.Parse(opts.platform)
		if err != nil {
			_, _ = fmt.Fprintf(dockerCli.Err(), "Invalid platform %s", opts.platform)
			return err
		}
		platform = &p

		printNote(dockerCli, `Using --platform pushes only the specified platform manifest of a multi-platform image index.
Other components, like attestations, will not be included.
To push the complete multi-platform image, remove the --platform flag.
`)
	}

	ref, err := reference.ParseNormalizedNamed(opts.remote)
	switch {
	case err != nil:
		return err
	case opts.all && !reference.IsNameOnly(ref):
		return errors.New("tag can't be used with --all-tags/-a")
	case !opts.all && reference.IsNameOnly(ref):
		ref = reference.TagNameOnly(ref)
		if tagged, ok := ref.(reference.Tagged); ok && !opts.quiet {
			_, _ = fmt.Fprintf(dockerCli.Out(), "Using default tag: %s\n", tagged.Tag())
		}
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}

	// Resolve the Auth config relevant for this server
	authConfig := command.ResolveAuthConfig(dockerCli.ConfigFile(), repoInfo.Index)
	encodedAuth, err := registrytypes.EncodeAuthConfig(authConfig)
	if err != nil {
		return err
	}
	requestPrivilege := command.RegistryAuthenticationPrivilegedFunc(dockerCli, repoInfo.Index, "push")
	options := image.PushOptions{
		All:           opts.all,
		RegistryAuth:  encodedAuth,
		PrivilegeFunc: requestPrivilege,
		Platform:      platform,
	}

	responseBody, err := dockerCli.Client().ImagePush(ctx, reference.FamiliarString(ref), options)
	if err != nil {
		return err
	}

	defer func() {
		for _, note := range notes {
			fmt.Fprintln(dockerCli.Err(), "")
			printNote(dockerCli, note)
		}
	}()

	defer responseBody.Close()
	if !opts.untrusted {
		// TODO PushTrustedReference currently doesn't respect `--quiet`
		return PushTrustedReference(dockerCli, repoInfo, ref, authConfig, responseBody)
	}

	if opts.quiet {
		err = jsonmessage.DisplayJSONMessagesToStream(responseBody, streams.NewOut(io.Discard), handleAux(dockerCli.Out()))
		if err == nil {
			fmt.Fprintln(dockerCli.Out(), ref.String())
		}
		return err
	}
	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), handleAux(dockerCli.Out()))
}

var notes []string

func handleAux(out *streams.Out) func(jm jsonmessage.JSONMessage) {
	return func(jm jsonmessage.JSONMessage) {
		b := []byte(*jm.Aux)

		var stripped auxprogress.ManifestPushedInsteadOfIndex
		err := json.Unmarshal(b, &stripped)
		if err == nil && stripped.ManifestPushedInsteadOfIndex {
			highlightColor := aec.NewBuilder(aec.GreenF, aec.Bold)

			note := fmt.Sprintf("Not all multiplatform-content is present, pushing single-platform image.")
			note += "\nNo platform selected, using host platform " + highlightColor.ANSI.Apply(stripped.SelectedManifest.Platform.OS+"/"+stripped.SelectedManifest.Platform.Architecture+"/"+stripped.SelectedManifest.Platform.Variant+"\n\n")
			headerColor := aec.NewBuilder(aec.DefaultF, aec.Bold).ANSI
			topNameColor := aec.NewBuilder(aec.BlueF, aec.Bold).ANSI
			normalColor := aec.NewBuilder(aec.DefaultF).ANSI
			untaggedColor := aec.NewBuilder(aec.Faint).ANSI
			// Print images
			columns := []imgColumn{
				{
					Title: "Image",
					Align: alignLeft,
					Width: 10,
				},
				{
					Title: "ID",
					Align: alignLeft,
					Width: 12,
					DetailsValue: func(d *rowDetails) string {
						return stringid.TruncateID(d.ID)
					},
				},
				{
					Title: "Disk usage",
					Align: alignRight,
					Width: 10,
					DetailsValue: func(d *rowDetails) string {
						return d.DiskUsage
					},
				},
				{
					Title: "Content size",
					Align: alignRight,
					Width: 12,
					DetailsValue: func(d *rowDetails) string {
						return d.ContentSize
					},
				},
			}

			imageRows, spacing := buildTableRows([]imagetypes.Summary{*stripped.ImageSummary})
			for i, child := range imageRows[0].Children {
				if child.Platform == stripped.SelectedManifest.Platform.OS+"/"+stripped.SelectedManifest.Platform.Architecture+"/"+stripped.SelectedManifest.Platform.Variant {
					imageRows[0].Children[i].Highlight = true
				}
			}

			_, width := out.GetTtySize()
			columns = formatColumnsForOutput(int(width), columns, imageRows)

			table := imageTreeTable{
				columns:        columns,
				headerColor:    headerColor,
				indexNameColor: topNameColor,
				untaggedColor:  untaggedColor,
				normalColor:    normalColor,
				spacing:        spacing,
			}

			treeB := bytes.Buffer{}
			table.printTable(&treeB, imageRows)
			note += treeB.String()
			notes = append(notes, note)
		}

		var missing auxprogress.ContentMissing
		err = json.Unmarshal(b, &missing)
		if err == nil && missing.ContentMissing {
			note := `You're trying to push a manifest list/index which 
				references multiple platform specific manifests, but not all of them are available locally
				or available to the remote repository.

				Make sure you have all the referenced content and try again.

				You can also push only a single platform specific manifest directly by specifying the platform you want to push with the --platform flag.`
			notes = append(notes, note)
		}
	}
}

func printNote(dockerCli command.Cli, format string, args ...any) {
	if dockerCli.Err().IsTerminal() {
		format = strings.ReplaceAll(format, "--platform", aec.Bold.Apply("--platform"))
	}

	header := " Info -> "
	padding := len(header)
	if dockerCli.Err().IsTerminal() {
		padding = len("i Info > ")
		header = aec.Bold.Apply(aec.LightCyanB.Apply(aec.BlackF.Apply("i")) + " " + aec.LightCyanF.Apply("Info → "))
	}

	_, _ = fmt.Fprint(dockerCli.Err(), header)
	s := fmt.Sprintf(format, args...)
	for idx, line := range strings.Split(s, "\n") {
		if idx > 0 {
			_, _ = fmt.Fprint(dockerCli.Err(), strings.Repeat(" ", padding))
		}
		_, _ = fmt.Fprintln(dockerCli.Err(), aec.Italic.Apply(line))
	}
}
