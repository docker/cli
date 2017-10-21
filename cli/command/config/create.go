package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/system"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

type createOptions struct {
	name   string
	file   string
	labels opts.ListOpts
}

func newConfigCreateCommand(dockerCli command.Cli) *cobra.Command {
	createOpts := createOptions{
		labels: opts.NewListOpts(opts.ValidateEnv),
	}

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] CONFIG file|-",
		Short: "Create a configuration file from a file, directory or STDIN",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				createOpts.file = args[0]
			} else if len(args) > 1 {
				createOpts.name = args[0]
				createOpts.file = args[1]
			}
			return runConfigCreate(dockerCli, createOpts)
		},
	}
	flags := cmd.Flags()
	flags.VarP(&createOpts.labels, "label", "l", "Config labels")

	return cmd
}

func runConfigCreate(dockerCli command.Cli, options createOptions) error {

	var in io.Reader = dockerCli.In()
	labels := opts.ConvertKVStringsToMap(options.labels.GetAll())
	if options.file == "-" {
		return configCreate(dockerCli, in, options.name, labels)
	}
	info, err := os.Stat(options.file)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if options.name != "" {
			return fmt.Errorf("cannot give a config name for a directory path")
		}
		files, err := ioutil.ReadDir(options.file)
		if err != nil {
			return fmt.Errorf("error listing files in %s: %v", options.file, err)
		}
		for _, file := range files {
			filePath := path.Join(options.file, file.Name())
			if file.Mode().IsRegular() {
				if err = configCreateFromFile(dockerCli, filePath, file.Name(), labels); err != nil {
					return err
				}
			}
		}
	} else {
		return configCreateFromFile(dockerCli, options.file, options.name, labels)
	}
	return nil
}

func configCreateFromFile(dockerCli command.Cli, fileName string, configName string, labels map[string]string) error {

	file, err := system.OpenSequential(fileName)
	defer file.Close()
	if err != nil {
		return err
	}
	return configCreate(dockerCli, file, configName, labels)

}

func configCreate(dockerCli command.Cli, in io.Reader, configName string, labels map[string]string) error {
	client := dockerCli.Client()

	ctx := context.Background()
	configData, err := ioutil.ReadAll(in)
	if err != nil {
		return errors.Errorf("Error reading config %q content: %v", configName, err)
	}

	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name:   configName,
			Labels: labels,
		},
		Data: configData,
	}

	r, err := client.ConfigCreate(ctx, spec)
	if err != nil {
		return err
	}

	fmt.Fprintln(dockerCli.Out(), r.ID)
	return nil
}
