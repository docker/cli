// This file is intended for use with "go run"; it isn't really part of the package.

// +build docsgen

package main

import (
	"log"
	"os"

	clidocstool "github.com/docker/cli-docs-tool"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const defaultSourcePath = "docs/reference/commandline/"

type options struct {
	source string
	target string
}

func gen(opts *options) error {
	log.SetFlags(0)

	dockerCLI, err := command.NewDockerCli()
	if err != nil {
		return err
	}
	cmd := &cobra.Command{
		Use:   "docker [OPTIONS] COMMAND [ARG...]",
		Short: "The base command for the Docker CLI.",
	}
	commands.AddCommands(cmd, dockerCLI)

	c, err := clidocstool.New(clidocstool.Options{
		Root:      cmd,
		SourceDir: opts.source,
		TargetDir: opts.target,
		Plugin:    false,
	})
	if err != nil {
		return err
	}

	return c.GenYamlTree(cmd)
}

func run() error {
	opts := &options{}
	flags := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flags.StringVar(&opts.source, "source", defaultSourcePath, "Docs source folder")
	flags.StringVar(&opts.target, "target", defaultSourcePath, "Docs target folder")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	return gen(opts)
}

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR: %+v", err)
		os.Exit(1)
	}
}
