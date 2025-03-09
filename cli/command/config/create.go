package config

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/swarm"
	"github.com/moby/sys/sequential"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// CreateOptions specifies some options that are used when creating a config.
type CreateOptions struct {
	Name           string
	TemplateDriver string
	File           string
	Labels         opts.ListOpts
}

func newConfigCreateCommand(dockerCli command.Cli) *cobra.Command {
	createOpts := CreateOptions{
		Labels: opts.NewListOpts(opts.ValidateLabel),
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] CONFIG file|-",
		Short: "Create a config from a file or STDIN",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			createOpts.Name = args[0]
			createOpts.File = args[1]
			return RunConfigCreate(cmd.Context(), dockerCli, createOpts)
		},
		ValidArgsFunction: completion.NoComplete,
	}
	flags := cmd.Flags()
	flags.VarP(&createOpts.Labels, "label", "l", "Config labels")
	flags.StringVar(&createOpts.TemplateDriver, "template-driver", "", "Template driver")
	flags.SetAnnotation("template-driver", "version", []string{"1.37"})

	return cmd
}

// RunConfigCreate creates a config with the given options.
func RunConfigCreate(ctx context.Context, dockerCLI command.Cli, options CreateOptions) error {
	apiClient := dockerCLI.Client()

	configData, err := readConfigData(dockerCLI.In(), options.File)
	if err != nil {
		return errors.Errorf("Error reading content from %q: %v", options.File, err)
	}

	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name:   options.Name,
			Labels: opts.ConvertKVStringsToMap(options.Labels.GetAll()),
		},
		Data: configData,
	}
	if options.TemplateDriver != "" {
		spec.Templating = &swarm.Driver{
			Name: options.TemplateDriver,
		}
	}
	r, err := apiClient.ConfigCreate(ctx, spec)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), r.ID)
	return nil
}

// maxConfigSize is the maximum byte length of the [swarm.ConfigSpec.Data] field,
// as defined by [MaxConfigSize] in SwarmKit.
//
// [MaxConfigSize]: https://pkg.go.dev/github.com/moby/swarmkit/v2@v2.0.0-20250103191802-8c1959736554/manager/controlapi#MaxConfigSize
const maxConfigSize = 1000 * 1024 // 1000KB

// readConfigData reads the config from either stdin or the given fileName.
//
// It reads up to twice the maximum size of the config ([maxConfigSize]),
// just in case swarm's limit changes; this is only a safeguard to prevent
// reading arbitrary files into memory.
func readConfigData(in io.Reader, fileName string) (data []byte, retErr error) {
	if fileName != "" {
		defer func() {
			if retErr != nil {
				retErr = fmt.Errorf("error reading from %s: %w", fileName, retErr)
			} else if len(data) == 0 {
				retErr = fmt.Errorf("error reading from %s: data is empty", fileName)
			}
		}()
	}

	switch fileName {
	case "-":
		fileName = "STDIN"
		return io.ReadAll(io.LimitReader(in, 2*maxConfigSize))
	case "":
		return nil, errors.New("config file is required")
	default:
		// Open file with [FILE_FLAG_SEQUENTIAL_SCAN] on Windows, which
		// prevents Windows from aggressively caching it. We expect this
		// file to be only read once. Given that this is expected to be
		// a small file, this may not be a significant optimization, so
		// we could choose to omit this, and use a regular [os.Open].
		//
		// [FILE_FLAG_SEQUENTIAL_SCAN]: https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilea#FILE_FLAG_SEQUENTIAL_SCAN
		f, err := sequential.Open(fileName)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(io.LimitReader(f, 2*maxConfigSize))
	}
}
