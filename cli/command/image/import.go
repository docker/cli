package image

import (
	"context"
	"os"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	dockeropts "github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/spf13/cobra"
)

type importOptions struct {
	source    string
	reference string
	changes   dockeropts.ListOpts
	message   string
	platform  string
}

// NewImportCommand creates a new `docker import` command
func NewImportCommand(dockerCli command.Cli) *cobra.Command {
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
			return runImport(cmd.Context(), dockerCli, options)
		},
		Annotations: map[string]string{
			"aliases": "docker image import, docker import",
		},
	}

	flags := cmd.Flags()

	options.changes = dockeropts.NewListOpts(nil)
	flags.VarP(&options.changes, "change", "c", "Apply Dockerfile instruction to the created image")
	flags.StringVarP(&options.message, "message", "m", "", "Set commit message for imported image")
	command.AddPlatformFlag(flags, &options.platform)
	_ = cmd.RegisterFlagCompletionFunc("platform", completion.Platforms)

	return cmd
}

func runImport(ctx context.Context, dockerCli command.Cli, options importOptions) error {
	var source image.ImportSource
	switch {
	case options.source == "-":
		// import from STDIN
		source = image.ImportSource{
			Source:     dockerCli.In(),
			SourceName: options.source,
		}
	case strings.HasPrefix(options.source, "https://"), strings.HasPrefix(options.source, "http://"):
		// import from a remote source (handled by the daemon)
		source = image.ImportSource{
			SourceName: options.source,
		}
	default:
		// import from a local file
		file, err := os.Open(options.source)
		if err != nil {
			return err
		}
		defer file.Close()
		source = image.ImportSource{
			Source:     file,
			SourceName: "-",
		}
	}

	responseBody, err := dockerCli.Client().ImageImport(ctx, source, options.reference, image.ImportOptions{
		Message:  options.message,
		Changes:  options.changes.GetAll(),
		Platform: options.platform,
	})
	if err != nil {
		return err
	}
	defer responseBody.Close()

	return jsonmessage.DisplayJSONMessagesToStream(responseBody, dockerCli.Out(), nil)
}
