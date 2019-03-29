package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/commands"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const descriptionSourcePath = "docs/reference/commandline/"

func generateCliYaml(opts *options) error {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		return err
	}
	cmd := &cobra.Command{Use: "docker"}
	commands.AddCommands(cmd, dockerCli)
	disableFlagsInUseLine(cmd)
	source := filepath.Join(opts.source, descriptionSourcePath)
	if err := loadLongDescription(cmd, source); err != nil {
		return err
	}

	cmd.DisableAutoGenTag = true
	return GenYamlTree(cmd, opts.target)
}

func disableFlagsInUseLine(cmd *cobra.Command) {
	visitAll(cmd, func(ccmd *cobra.Command) {
		// do not add a `[flags]` to the end of the usage line.
		ccmd.DisableFlagsInUseLine = true
	})
}

// visitAll will traverse all commands from the root.
// This is different from the VisitAll of cobra.Command where only parents
// are checked.
func visitAll(root *cobra.Command, fn func(*cobra.Command)) {
	for _, cmd := range root.Commands() {
		visitAll(cmd, fn)
	}
	fn(root)
}

func loadLongDescription(cmd *cobra.Command, path ...string) error {
	for _, cmd := range cmd.Commands() {
		if cmd.Name() == "" {
			continue
		}
		fullpath := filepath.Join(path[0], strings.Join(append(path[1:], cmd.Name()), "_")+".md")

		if cmd.HasSubCommands() {
			loadLongDescription(cmd, path[0], cmd.Name())
		}

		if _, err := os.Stat(fullpath); err != nil {
			log.Printf("WARN: %s does not exist, skipping\n", fullpath)
			continue
		}

		content, err := ioutil.ReadFile(fullpath)
		if err != nil {
			return err
		}
		description, examples := parseMDContent(string(content))
		cmd.Long = description
		cmd.Example = examples
	}
	return nil
}

type options struct {
	source string
	target string
}

func parseArgs() (*options, error) {
	opts := &options{}
	cwd, _ := os.Getwd()
	flags := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	flags.StringVar(&opts.source, "root", cwd, "Path to project root")
	flags.StringVar(&opts.target, "target", "/tmp", "Target path for generated yaml files")
	err := flags.Parse(os.Args[1:])
	return opts, err
}

func main() {
	opts, err := parseArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}
	fmt.Printf("Project root: %s\n", opts.source)
	fmt.Printf("Generating yaml files into %s\n", opts.target)
	if err := generateCliYaml(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate yaml files: %s\n", err.Error())
	}
}
