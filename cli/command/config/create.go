package config

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
	templateDriver string
	file           string
	labels         opts.ListOpts
}

func newConfigCreateCommand(dockerCli command.Cli) *cobra.Command {
	createOpts := createOptions{
		labels: opts.NewListOpts(opts.ValidateEnv),
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] CONFIG file|-",
		Short: "Create a config from a file or STDIN",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			createOpts.name = args[0]
			createOpts.file = args[1]
			return runConfigCreate(dockerCli, createOpts)
		},
	}
	flags := cmd.Flags()
	flags.VarP(&createOpts.labels, "label", "l", "Config labels")
	flags.StringVar(&createOpts.templateDriver, "template-driver", "", "Template driver")
	flags.SetAnnotation("driver", "version", []string{"1.37"})

	return cmd
}

func runConfigCreate(dockerCli command.Cli, options createOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	var in io.Reader = dockerCli.In()
	if options.file != "-" {
		file, err := system.OpenSequential(options.file)
		if err != nil {
			return err
		}
		in = file
		defer file.Close()
	}

	configData, err := ioutil.ReadAll(in)
	if err != nil {
		return errors.Errorf("Error reading content from %q: %v", options.file, err)
	}

	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name:   options.name,
			Labels: opts.ConvertKVStringsToMap(options.labels.GetAll()),
		},
		Data: configData,
	}
	if options.templateDriver != "" {
		spec.Templating = &swarm.Driver{
			Name: options.templateDriver,
		}
	}
	r, err := client.ConfigCreate(ctx, spec)
	if err != nil {
		return err
	}

	fmt.Fprintln(dockerCli.Out(), r.ID)
	return nil
}
