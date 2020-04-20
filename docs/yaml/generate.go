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
	cmd := &cobra.Command{
		Use:   "docker [OPTIONS] COMMAND [ARG...]",
		Short: "The base command for the Docker CLI.",
	}
	commands.AddCommands(cmd, dockerCli)
	disableFlagsInUseLine(cmd)
	source := filepath.Join(opts.source, descriptionSourcePath)
	fmt.Println("Markdown source:", source)
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

func loadLongDescription(parentCmd *cobra.Command, path string) error {
	for _, cmd := range parentCmd.Commands() {
		if cmd.HasSubCommands() {
			if err := loadLongDescription(cmd, path); err != nil {
				return err
			}
		}
		name := cmd.CommandPath()
		log.Println("INFO: Generating docs for", name)
		if i := strings.Index(name, " "); i >= 0 {
			// remove root command / binary name
			name = name[i+1:]
		}
		if name == "" {
			continue
		}
		mdFile := strings.ReplaceAll(name, " ", "_") + ".md"
		fullPath := filepath.Join(path, mdFile)
		content, err := ioutil.ReadFile(fullPath)
		if os.IsNotExist(err) {
			log.Printf("WARN: %s does not exist, skipping\n", mdFile)
			continue
		}
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
		log.Println(err)
	}
	fmt.Println("Project root:   ", opts.source)
	fmt.Println("YAML output dir:", opts.target)
	if err := generateCliYaml(opts); err != nil {
		log.Println("Failed to generate yaml files:", err)
	}
}
