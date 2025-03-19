// This file is intended for use with "go run"; it isn't really part of the package.

//go:build manpages

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	clidocstool "github.com/docker/cli-docs-tool"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"
)

const defaultSourcePath = "man/src/"

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

	clientOpts, _ := cli.SetupRootCommand(cmd)
	if err := dockerCLI.Initialize(clientOpts); err != nil {
		return err
	}
	commands.AddCommands(cmd, dockerCLI)
	// TODO(thaJeztah): cli-docs-tool should already be able to do this, but assumes source-files are not in subdirectories (it looks for `src/docker_command_subcommand.md`)
	if err := loadLongDescription(cmd, opts.source); err != nil {
		return err
	}

	c, err := clidocstool.New(clidocstool.Options{
		Root:      cmd,
		SourceDir: opts.source,
		TargetDir: opts.target,
		ManHeader: &doc.GenManHeader{
			Title:   "DOCKER",
			Section: "1",
			Source:  "Docker Community",
			Manual:  "Docker User Manuals",
		},
		Plugin: false,
	})
	if err != nil {
		return err
	}
	fmt.Println("Manpage source folder:", opts.source)
	fmt.Println("Generating man pages into", opts.target)
	return c.GenManTree(cmd)
}

func loadLongDescription(parentCommand *cobra.Command, path string) error {
	for _, cmd := range parentCommand.Commands() {
		cmd.DisableFlagsInUseLine = true
		if cmd.Name() == "" {
			continue
		}
		fullpath := filepath.Join(path, cmd.Name()+".md")

		if cmd.HasSubCommands() {
			if err := loadLongDescription(cmd, filepath.Join(path, cmd.Name())); err != nil {
				return err
			}
		}

		if _, err := os.Stat(fullpath); err != nil {
			log.Printf("WARN: %s does not exist, skipping\n", fullpath)
			continue
		}

		log.Printf("INFO: %s found\n", fullpath)
		content, err := os.ReadFile(fullpath)
		if err != nil {
			return err
		}
		cmd.Long = string(content)

		fullpath = filepath.Join(path, cmd.Name()+"-example.md")
		if _, err := os.Stat(fullpath); err != nil {
			continue
		}

		content, err = os.ReadFile(fullpath)
		if err != nil {
			return err
		}
		cmd.Example = string(content)
	}
	return nil
}

func run() error {
	opts := &options{}
	flags := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flags.StringVar(&opts.source, "source", defaultSourcePath, "Manpage source folder")
	flags.StringVar(&opts.target, "target", "/tmp", "Target path for generated man pages")
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
