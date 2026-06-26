package secret

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

type createOptions struct {
	name           string
	driver         string
	templateDriver string
	file           string
	labels         opts.ListOpts
}

func newSecretCreateCommand(dockerCLI command.Cli) *cobra.Command {
	options := createOptions{
		labels: opts.NewListOpts(opts.ValidateLabel),
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] SECRET [file|-]",
		Short: "Create a secret from a file or STDIN as content",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.name = args[0]
			if len(args) == 2 {
				options.file = args[1]
			}
			return runSecretCreate(cmd.Context(), dockerCLI, options)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0:
				// No completion for the first argument, which is the name for
				// the new secret, but if a non-empty name is given, we return
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
	flags.VarP(&options.labels, "label", "l", "Secret labels")
	flags.StringVarP(&options.driver, "driver", "d", "", "Secret driver")
	flags.SetAnnotation("driver", "version", []string{"1.31"})
	flags.StringVar(&options.templateDriver, "template-driver", "", "Template driver")
	flags.SetAnnotation("template-driver", "version", []string{"1.37"})

	return cmd
}

func runSecretCreate(ctx context.Context, dockerCLI command.Cli, options createOptions) error {
	apiClient := dockerCLI.Client()

	var secretData []byte
	if options.driver != "" {
		if options.file != "" {
			return errors.New("when using secret driver secret data must be empty")
		}
	} else {
		var err error
		secretData, err = readSecretData(dockerCLI.In(), options.file)
		if err != nil {
			return err
		}
	}

	spec := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name:   options.name,
			Labels: opts.ConvertKVStringsToMap(options.labels.GetSlice()),
		},
		Data: secretData,
	}
	if options.driver != "" {
		spec.Driver = &swarm.Driver{
			Name: options.driver,
		}
	}
	if options.templateDriver != "" {
		spec.Templating = &swarm.Driver{
			Name: options.templateDriver,
		}
	}
	r, err := apiClient.SecretCreate(ctx, client.SecretCreateOptions{
		Spec: spec,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), r.ID)
	return nil
}

// maxSecretSize is the maximum byte length of the [swarm.SecretSpec.Data] field,
// as defined by [MaxSecretSize] in SwarmKit.
//
// [MaxSecretSize]: https://pkg.go.dev/github.com/moby/swarmkit/v2@v2.0.0-20250103191802-8c1959736554/api/validation#MaxSecretSize
const maxSecretSize = 500 * 1024 // 500KB

// readSecretData reads the secret from either stdin or the given fileName.
//
// It reads up to twice the maximum size of the secret ([maxSecretSize]),
// just in case swarm's limit changes; this is only a safeguard to prevent
// reading arbitrary files into memory.
func readSecretData(in io.Reader, fileName string) ([]byte, error) {
	switch fileName {
	case "-":
		data, err := io.ReadAll(io.LimitReader(in, 2*maxSecretSize))
		if err != nil {
			return nil, fmt.Errorf("error reading from STDIN: %w", err)
		}
		if len(data) == 0 {
			return nil, errors.New("error reading from STDIN: data is empty")
		}
		return data, nil
	case "":
		return nil, errors.New("secret file is required")
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
		data, err := io.ReadAll(io.LimitReader(f, 2*maxSecretSize))
		if err != nil {
			return nil, fmt.Errorf("error reading from %s: %w", fileName, err)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("error reading from %s: data is empty", fileName)
		}
		return data, nil
	}
}
