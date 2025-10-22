package config

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/moby/sys/sequential"
	"github.com/spf13/cobra"
)

// createOptions specifies some options that are used when creating a config.
type createOptions struct {
	name           string
	templateDriver string
	file           string
	labels         opts.ListOpts
}

func newConfigCreateCommand(dockerCLI command.Cli) *cobra.Command {
	createOpts := createOptions{
		labels: opts.NewListOpts(opts.ValidateLabel),
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] CONFIG file|-",
		Short: "Create a config from a file or STDIN",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			createOpts.name = args[0]
			createOpts.file = args[1]
			return runCreate(cmd.Context(), dockerCLI, createOpts)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0:
				// No completion for the first argument, which is the name for
				// the new config, but if a non-empty name is given, we return
				// it as completion to allow "tab"-ing to the next completion.
				return []string{toComplete}, cobra.ShellCompDirectiveNoFileComp
			case 1:
				// Second argument is either "-" or a file to load.
				//
				// TODO(thaJeztah): provide completion for "-".
				return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveDefault
			default:
				// Command only accepts two arguments.
				return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			}
		},
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.VarP(&createOpts.labels, "label", "l", "Config labels")
	flags.StringVar(&createOpts.templateDriver, "template-driver", "", "Template driver")
	_ = flags.SetAnnotation("template-driver", "version", []string{"1.37"})

	return cmd
}

// runCreate creates a config with the given options.
func runCreate(ctx context.Context, dockerCLI command.Cli, options createOptions) error {
	apiClient := dockerCLI.Client()

	configData, err := readConfigData(dockerCLI.In(), options.file)
	if err != nil {
		return fmt.Errorf("error reading content from %q: %v", options.file, err)
	}

	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name:   options.name,
			Labels: opts.ConvertKVStringsToMap(options.labels.GetSlice()),
		},
		Data: configData,
	}
	if options.templateDriver != "" {
		spec.Templating = &swarm.Driver{
			Name: options.templateDriver,
		}
	}
	r, err := apiClient.ConfigCreate(ctx, client.ConfigCreateOptions{
		Spec: spec,
	})
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
func readConfigData(in io.Reader, fileName string) ([]byte, error) {
	switch fileName {
	case "-":
		data, err := io.ReadAll(io.LimitReader(in, 2*maxConfigSize))
		if err != nil {
			return nil, fmt.Errorf("error reading from STDIN: %w", err)
		}
		if len(data) == 0 {
			return nil, errors.New("error reading from STDIN: data is empty")
		}
		return data, nil
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
			return nil, fmt.Errorf("error reading from %s: %w", fileName, err)
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, 2*maxConfigSize))
		if err != nil {
			return nil, fmt.Errorf("error reading from %s: %w", fileName, err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("error reading from %s: data is empty", fileName)
		}
		return data, nil
	}
}
