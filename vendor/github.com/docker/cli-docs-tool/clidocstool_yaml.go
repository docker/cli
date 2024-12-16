// Copyright 2017 cli-docs-tool authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clidocstool

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/docker/cli-docs-tool/annotation"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type cmdOption struct {
	Option          string
	Shorthand       string `yaml:",omitempty"`
	ValueType       string `yaml:"value_type,omitempty"`
	DefaultValue    string `yaml:"default_value,omitempty"`
	Description     string `yaml:",omitempty"`
	DetailsURL      string `yaml:"details_url,omitempty"` // DetailsURL contains an anchor-id or link for more information on this flag
	Deprecated      bool
	Hidden          bool
	MinAPIVersion   string `yaml:"min_api_version,omitempty"`
	Experimental    bool
	ExperimentalCLI bool
	Kubernetes      bool
	Swarm           bool
	OSType          string `yaml:"os_type,omitempty"`
}

type cmdDoc struct {
	Name             string      `yaml:"command"`
	SeeAlso          []string    `yaml:"parent,omitempty"`
	Version          string      `yaml:"engine_version,omitempty"`
	Aliases          string      `yaml:",omitempty"`
	Short            string      `yaml:",omitempty"`
	Long             string      `yaml:",omitempty"`
	Usage            string      `yaml:",omitempty"`
	Pname            string      `yaml:",omitempty"`
	Plink            string      `yaml:",omitempty"`
	Cname            []string    `yaml:",omitempty"`
	Clink            []string    `yaml:",omitempty"`
	Options          []cmdOption `yaml:",omitempty"`
	InheritedOptions []cmdOption `yaml:"inherited_options,omitempty"`
	Example          string      `yaml:"examples,omitempty"`
	Deprecated       bool
	Hidden           bool
	MinAPIVersion    string `yaml:"min_api_version,omitempty"`
	Experimental     bool
	ExperimentalCLI  bool
	Kubernetes       bool
	Swarm            bool
	OSType           string `yaml:"os_type,omitempty"`
}

// GenYamlTree creates yaml structured ref files for this command and all descendants
// in the directory given. This function may not work
// correctly if your command names have `-` in them. If you have `cmd` with two
// subcmds, `sub` and `sub-third`, and `sub` has a subcommand called `third`
// it is undefined which help output will be in the file `cmd-sub-third.1`.
func (c *Client) GenYamlTree(cmd *cobra.Command) error {
	emptyStr := func(string) string { return "" }
	if err := c.loadLongDescription(cmd, "yaml"); err != nil {
		return err
	}
	return c.genYamlTreeCustom(cmd, emptyStr)
}

// genYamlTreeCustom creates yaml structured ref files.
func (c *Client) genYamlTreeCustom(cmd *cobra.Command, filePrepender func(string) string) error {
	for _, sc := range cmd.Commands() {
		if !sc.Runnable() && !sc.HasAvailableSubCommands() {
			// skip non-runnable commands without subcommands
			// but *do* generate YAML for hidden and deprecated commands
			// the YAML will have those included as metadata, so that the
			// documentation repository can decide whether or not to present them
			continue
		}
		if err := c.genYamlTreeCustom(sc, filePrepender); err != nil {
			return err
		}
	}

	// always disable the addition of [flags] to the usage
	cmd.DisableFlagsInUseLine = true

	// The "root" command used in the generator is just a "stub", and only has a
	// list of subcommands, but not (e.g.) global options/flags. We should fix
	// that, so that the YAML file for the docker "root" command contains the
	// global flags.

	// Skip the root command altogether, to prevent generating a useless
	// YAML file for plugins.
	if c.plugin && !cmd.HasParent() {
		return nil
	}

	log.Printf("INFO: Generating YAML for %q", cmd.CommandPath())
	basename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".yaml"
	target := filepath.Join(c.target, basename)
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(target)); err != nil {
		return err
	}
	return c.genYamlCustom(cmd, f)
}

// genYamlCustom creates custom yaml output.
// nolint: gocyclo
func (c *Client) genYamlCustom(cmd *cobra.Command, w io.Writer) error {
	const (
		// shortMaxWidth is the maximum width for the "Short" description before
		// we force YAML to use multi-line syntax. The goal is to make the total
		// width fit within 80 characters. This value is based on 80 characters
		// minus the with of the field, colon, and whitespace ('short: ').
		shortMaxWidth = 73

		// longMaxWidth is the maximum width for the "Short" description before
		// we force YAML to use multi-line syntax. The goal is to make the total
		// width fit within 80 characters. This value is based on 80 characters
		// minus the with of the field, colon, and whitespace ('long: ').
		longMaxWidth = 74
	)

	// necessary to add inherited flags otherwise some
	// fields are not properly declared like usage
	cmd.Flags().AddFlagSet(cmd.InheritedFlags())

	cliDoc := cmdDoc{
		Name:       cmd.CommandPath(),
		Aliases:    strings.Join(getAliases(cmd), ", "),
		Short:      forceMultiLine(cmd.Short, shortMaxWidth),
		Long:       forceMultiLine(cmd.Long, longMaxWidth),
		Example:    cmd.Example,
		Deprecated: len(cmd.Deprecated) > 0,
		Hidden:     cmd.Hidden,
	}

	if len(cliDoc.Long) == 0 {
		cliDoc.Long = cliDoc.Short
	}

	if cmd.Runnable() {
		cliDoc.Usage = cmd.UseLine()
	}

	// check recursively to handle inherited annotations
	for curr := cmd; curr != nil; curr = curr.Parent() {
		if v, ok := curr.Annotations["version"]; ok && cliDoc.MinAPIVersion == "" {
			cliDoc.MinAPIVersion = v
		}
		if _, ok := curr.Annotations["experimental"]; ok && !cliDoc.Experimental {
			cliDoc.Experimental = true
		}
		if _, ok := curr.Annotations["experimentalCLI"]; ok && !cliDoc.ExperimentalCLI {
			cliDoc.ExperimentalCLI = true
		}
		if _, ok := curr.Annotations["kubernetes"]; ok && !cliDoc.Kubernetes {
			cliDoc.Kubernetes = true
		}
		if _, ok := curr.Annotations["swarm"]; ok && !cliDoc.Swarm {
			cliDoc.Swarm = true
		}
		if o, ok := curr.Annotations["ostype"]; ok && cliDoc.OSType == "" {
			cliDoc.OSType = o
		}
		if _, ok := cmd.Annotations[annotation.CodeDelimiter]; !ok {
			if cd, cok := curr.Annotations[annotation.CodeDelimiter]; cok {
				if cmd.Annotations == nil {
					cmd.Annotations = map[string]string{}
				}
				cmd.Annotations[annotation.CodeDelimiter] = cd
			}
		}
	}

	anchors := make(map[string]struct{})
	if a, ok := cmd.Annotations["anchors"]; ok && a != "" {
		for _, anchor := range strings.Split(a, ",") {
			anchors[anchor] = struct{}{}
		}
	}

	flags := cmd.NonInheritedFlags()
	if flags.HasFlags() {
		cliDoc.Options = genFlagResult(cmd, flags, anchors)
	}
	flags = cmd.InheritedFlags()
	if flags.HasFlags() {
		cliDoc.InheritedOptions = genFlagResult(cmd, flags, anchors)
	}

	if hasSeeAlso(cmd) {
		if cmd.HasParent() {
			parent := cmd.Parent()
			cliDoc.Pname = parent.CommandPath()
			cliDoc.Plink = strings.Replace(cliDoc.Pname, " ", "_", -1) + ".yaml"
			cmd.VisitParents(func(c *cobra.Command) {
				if c.DisableAutoGenTag {
					cmd.DisableAutoGenTag = c.DisableAutoGenTag
				}
			})
		}

		children := cmd.Commands()
		sort.Sort(byName(children))

		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			cliDoc.Cname = append(cliDoc.Cname, cliDoc.Name+" "+child.Name())
			cliDoc.Clink = append(cliDoc.Clink, strings.Replace(cliDoc.Name+"_"+child.Name(), " ", "_", -1)+".yaml")
		}
	}

	final, err := yaml.Marshal(&cliDoc)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if _, err := fmt.Fprintln(w, string(final)); err != nil {
		return err
	}
	return nil
}

func genFlagResult(cmd *cobra.Command, flags *pflag.FlagSet, anchors map[string]struct{}) []cmdOption {
	var (
		result []cmdOption
		opt    cmdOption
	)

	const (
		// shortMaxWidth is the maximum width for the "Short" description before
		// we force YAML to use multi-line syntax. The goal is to make the total
		// width fit within 80 characters. This value is based on 80 characters
		// minus the with of the field, colon, and whitespace ('  default_value: ').
		defaultValueMaxWidth = 64

		// longMaxWidth is the maximum width for the "Short" description before
		// we force YAML to use multi-line syntax. The goal is to make the total
		// width fit within 80 characters. This value is based on 80 characters
		// minus the with of the field, colon, and whitespace ('  description: ').
		descriptionMaxWidth = 66
	)

	flags.VisitAll(func(flag *pflag.Flag) {
		opt = cmdOption{
			Option:     flag.Name,
			ValueType:  flag.Value.Type(),
			Deprecated: len(flag.Deprecated) > 0,
			Hidden:     flag.Hidden,
		}

		var defval string
		if v, ok := flag.Annotations[annotation.DefaultValue]; ok && len(v) > 0 {
			defval = v[0]
			if cd, ok := flag.Annotations[annotation.CodeDelimiter]; ok {
				defval = strings.ReplaceAll(defval, cd[0], "`")
			} else if cd, ok := cmd.Annotations[annotation.CodeDelimiter]; ok {
				defval = strings.ReplaceAll(defval, cd, "`")
			}
		} else {
			defval = flag.DefValue
		}
		opt.DefaultValue = forceMultiLine(defval, defaultValueMaxWidth)

		usage := flag.Usage
		if cd, ok := flag.Annotations[annotation.CodeDelimiter]; ok {
			usage = strings.ReplaceAll(usage, cd[0], "`")
		} else if cd, ok := cmd.Annotations[annotation.CodeDelimiter]; ok {
			usage = strings.ReplaceAll(usage, cd, "`")
		}
		opt.Description = forceMultiLine(usage, descriptionMaxWidth)

		if v, ok := flag.Annotations[annotation.ExternalURL]; ok && len(v) > 0 {
			opt.DetailsURL = strings.TrimPrefix(v[0], "https://docs.docker.com")
		} else if _, ok = anchors[flag.Name]; ok {
			opt.DetailsURL = "#" + flag.Name
		}

		// Todo, when we mark a shorthand is deprecated, but specify an empty message.
		// The flag.ShorthandDeprecated is empty as the shorthand is deprecated.
		// Using len(flag.ShorthandDeprecated) > 0 can't handle this, others are ok.
		if !(len(flag.ShorthandDeprecated) > 0) && len(flag.Shorthand) > 0 {
			opt.Shorthand = flag.Shorthand
		}
		if _, ok := flag.Annotations["experimental"]; ok {
			opt.Experimental = true
		}
		if _, ok := flag.Annotations["deprecated"]; ok {
			opt.Deprecated = true
		}
		if v, ok := flag.Annotations["version"]; ok {
			opt.MinAPIVersion = v[0]
		}
		if _, ok := flag.Annotations["experimentalCLI"]; ok {
			opt.ExperimentalCLI = true
		}
		if _, ok := flag.Annotations["kubernetes"]; ok {
			opt.Kubernetes = true
		}
		if _, ok := flag.Annotations["swarm"]; ok {
			opt.Swarm = true
		}

		// Note that the annotation can have multiple ostypes set, however, multiple
		// values are currently not used (and unlikely will).
		//
		// To simplify usage of the os_type property in the YAML, and for consistency
		// with the same property for commands, we're only using the first ostype that's set.
		if ostypes, ok := flag.Annotations["ostype"]; ok && len(opt.OSType) == 0 && len(ostypes) > 0 {
			opt.OSType = ostypes[0]
		}

		result = append(result, opt)
	})

	return result
}

// forceMultiLine appends a newline (\n) to strings that are longer than max
// to force the yaml lib to use block notation (https://yaml.org/spec/1.2/spec.html#Block)
// instead of a single-line string with newlines and tabs encoded("string\nline1\nline2").
//
// This makes the generated YAML more readable, and easier to review changes.
// max can be used to customize the width to keep the whole line < 80 chars.
func forceMultiLine(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max && !strings.Contains(s, "\n") {
		s = s + "\n"
	}
	return s
}

// Small duplication for cobra utils
func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

type byName []*cobra.Command

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
