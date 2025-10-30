package image

import (
	"context"
	"os"
	"strings"

	"github.com/containerd/platforms"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/internal/jsonstream"
	dockeropts "github.com/docker/cli/opts"
	"github.com/moby/moby/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

type importOptions struct {
	source    string
	reference string
	changes   dockeropts.ListOpts
	message   string
	platform  string
}

// newImportCommand creates a new "docker image import" command.
func newImportCommand(dockerCLI command.Cli) *cobra.Command {
	var options importOptions

	cmd := &cobra.Command{
		Use:   "import [OPTIONS] file|URL|- [REPOSITORY[:TAG]]",
		Short: "Import the contents from a tarball to create a filesystem image",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.source = args[0]
			if len(args) > 1 {
				options.reference = args[1]
			}
			return runImport(cmd.Context(), dockerCLI, options)
		},
		Annotations: map[string]string{
			"aliases": "docker image import, docker import",
		},
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()

	options.changes = dockeropts.NewListOpts(nil)
	flags.VarP(&options.changes, "change", "c", "Apply Dockerfile instruction to the created image")
	flags.StringVarP(&options.message, "message", "m", "", "Set commit message for imported image")
	flags.StringVar(&options.platform, "platform", os.Getenv("DOCKER_DEFAULT_PLATFORM"), "Set platform if server is multi-platform capable")
	_ = flags.SetAnnotation("platform", "version", []string{"1.32"})
	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms())

	return cmd
}

func runImport(ctx context.Context, dockerCli command.Cli, options importOptions) error {
	var source client.ImageImportSource
	switch {
	case options.source == "-":
		// import from STDIN
		source = client.ImageImportSource{
			Source:     dockerCli.In(),
			SourceName: options.source,
		}
	case strings.HasPrefix(options.source, "https://"), strings.HasPrefix(options.source, "http://"):
		// import from a remote source (handled by the daemon)
		source = client.ImageImportSource{
			SourceName: options.source,
		}
	default:
		// import from a local file
		file, err := os.Open(options.source)
		if err != nil {
			return err
		}
		defer file.Close()
		source = client.ImageImportSource{
			Source:     file,
			SourceName: "-",
		}
	}

	// TODO(thaJeztah): add a platform option-type / flag-type.
	var ociPlatform ocispec.Platform
	if options.platform != "" {
		var err error
		ociPlatform, err = platforms.Parse(options.platform)
		if err != nil {
			return err
		}
	}

	responseBody, err := dockerCli.Client().ImageImport(ctx, source, options.reference, client.ImageImportOptions{
		Message:  options.message,
		Changes:  options.changes.GetSlice(),
		Platform: ociPlatform,
	})
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonstream.Display(ctx, responseBody, dockerCli.Out())
}
