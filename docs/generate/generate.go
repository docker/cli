// This file is intended for use with "go run"; it isn't really part of the package.

//go:build docsgen
// +build docsgen

package main

import (
	"log"
	"os"

	clidocstool "github.com/docker/cli-docs-tool"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const defaultSourcePath = "docs/reference/commandline/"

type options struct {
	source  string
	target  string
	formats []string
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
	clientOpts, _ := cli.SetupRootCommand(cmd)
	if err := dockerCLI.Initialize(clientOpts); err != nil {
		return err
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

	for _, format := range opts.formats {
		switch format {
		case "md":
			if err = c.GenMarkdownTree(cmd); err != nil {
				return err
			}
		case "yaml":
			if err = c.GenYamlTree(cmd); err != nil {
				return err
			}
		default:
			return errors.Errorf("unknown format %q", format)
		}
	}

	return nil
}

func run() error {
	opts := &options{}
	flags := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flags.StringVar(&opts.source, "source", defaultSourcePath, "Docs source folder")
	flags.StringVar(&opts.target, "target", defaultSourcePath, "Docs target folder")
	flags.StringSliceVar(&opts.formats, "formats", []string{}, "Format (md, yaml)")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	if len(opts.formats) == 0 {
		return errors.New("Docs format required")
	}
	return gen(opts)
}

func main() {
	if err := run(); err != nil {
		log.Printf("ERROR: %+v", err)
		os.Exit(1)
	}
}
