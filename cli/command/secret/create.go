package secret

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/system"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type createOptions struct {
	name           string
	driver         string
	templateDriver string
	file           string
	labels         opts.ListOpts
}

func newSecretCreateCommand(dockerCli command.Cli) *cobra.Command {
	options := createOptions{
		labels: opts.NewListOpts(opts.ValidateEnv),
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
			return runSecretCreate(dockerCli, options)
		},
	}
	flags := cmd.Flags()
	flags.VarP(&options.labels, "label", "l", "Secret labels")
	flags.StringVarP(&options.driver, "driver", "d", "", "Secret driver")
	flags.SetAnnotation("driver", "version", []string{"1.31"})
	flags.StringVar(&options.templateDriver, "template-driver", "", "Template driver")
	flags.SetAnnotation("driver", "version", []string{"1.37"})

	return cmd
}

func runSecretCreate(dockerCli command.Cli, options createOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	if options.driver != "" && options.file != "" {
		return errors.Errorf("When using secret driver secret data must be empty")
	}

	secretData, err := readSecretData(dockerCli.In(), options.file)
	if err != nil {
		return errors.Errorf("Error reading content from %q: %v", options.file, err)
	}
	spec := swarm.SecretSpec{
		Annotations: swarm.Annotations{
			Name:   options.name,
			Labels: opts.ConvertKVStringsToMap(options.labels.GetAll()),
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
	r, err := client.SecretCreate(ctx, spec)
	if err != nil {
		return err
	}

	fmt.Fprintln(dockerCli.Out(), r.ID)
	return nil
}

func readSecretData(in io.ReadCloser, file string) ([]byte, error) {
	// Read secret value from external driver
	if file == "" {
		return nil, nil
	}
	if file != "-" {
		var err error
		in, err = system.OpenSequential(file)
		if err != nil {
			return nil, err
		}
		defer in.Close()
	}
	data, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}
	return data, nil
}
