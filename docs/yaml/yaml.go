package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v2"
)

type cmdOption struct {
	Option          string
	Shorthand       string `yaml:",omitempty"`
	ValueType       string `yaml:"value_type,omitempty"`
	DefaultValue    string `yaml:"default_value,omitempty"`
	Description     string `yaml:",omitempty"`
	DetailsURL      string `yaml:"details_url,omitempty"` // DetailsURL contains an anchor-id or link for more information on this flag
	Deprecated      bool
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
	MinAPIVersion    string `yaml:"min_api_version,omitempty"`
	Experimental     bool
	ExperimentalCLI  bool
	Kubernetes       bool
	Swarm            bool
	OSType           string `yaml:"os_type,omitempty"`
}

// GenYamlTree creates yaml structured ref files
func GenYamlTree(cmd *cobra.Command, dir string) error {
	emptyStr := func(s string) string { return "" }
	return GenYamlTreeCustom(cmd, dir, emptyStr)
}

// GenYamlTreeCustom creates yaml structured ref files
func GenYamlTreeCustom(cmd *cobra.Command, dir string, filePrepender func(string) string) error {
	for _, c := range cmd.Commands() {
		if !c.Runnable() && !c.HasAvailableSubCommands() {
			// skip non-runnable commands without subcommands
			// but *do* generate YAML for hidden and deprecated commands
			// the YAML will have those included as metadata, so that the
			// documentation repository can decide whether or not to present them
			continue
		}
		if err := GenYamlTreeCustom(c, dir, filePrepender); err != nil {
			return err
		}
	}
	if !cmd.HasParent() {
		return nil
	}
	basename := strings.Replace(cmd.CommandPath(), " ", "_", -1) + ".yaml"
	filename := filepath.Join(dir, basename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.WriteString(f, filePrepender(filename)); err != nil {
		return err
	}
	return GenYamlCustom(cmd, f)
}

// GenYamlCustom creates custom yaml output
// nolint: gocyclo
func GenYamlCustom(cmd *cobra.Command, w io.Writer) error {
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

	cliDoc := cmdDoc{
		Name:       cmd.CommandPath(),
		Aliases:    strings.Join(cmd.Aliases, ", "),
		Short:      forceMultiLine(cmd.Short, shortMaxWidth),
		Long:       forceMultiLine(cmd.Long, longMaxWidth),
		Example:    cmd.Example,
		Deprecated: len(cmd.Deprecated) > 0,
	}

	if len(cliDoc.Long) == 0 {
		cliDoc.Long = cliDoc.Short
	}

	if cmd.Runnable() {
		cliDoc.Usage = cmd.UseLine()
	}

	// Check recursively so that, e.g., `docker stack ls` returns the same output as `docker stack`
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
	}

	anchors := make(map[string]struct{})
	if a, ok := cmd.Annotations["anchors"]; ok && a != "" {
		for _, anchor := range strings.Split(a, ",") {
			anchors[anchor] = struct{}{}
		}
	}

	flags := cmd.NonInheritedFlags()
	if flags.HasFlags() {
		cliDoc.Options = genFlagResult(flags, anchors)
	}
	flags = cmd.InheritedFlags()
	if flags.HasFlags() {
		cliDoc.InheritedOptions = genFlagResult(flags, anchors)
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

func genFlagResult(flags *pflag.FlagSet, anchors map[string]struct{}) []cmdOption {
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
			Option:       flag.Name,
			ValueType:    flag.Value.Type(),
			DefaultValue: forceMultiLine(flag.DefValue, defaultValueMaxWidth),
			Description:  forceMultiLine(flag.Usage, descriptionMaxWidth),
			Deprecated:   len(flag.Deprecated) > 0,
		}

		if v, ok := flag.Annotations["docs.external.url"]; ok && len(v) > 0 {
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

// applyDescriptionAndExamples fills in cmd.Long and cmd.Example with the
// "Description" and "Examples" H2 sections in  mdString (if present).
func applyDescriptionAndExamples(cmd *cobra.Command, mdString string) {
	sections := getSections(mdString)
	var (
		anchors []string
		md      string
	)
	if sections["description"] != "" {
		md, anchors = cleanupMarkDown(sections["description"])
		cmd.Long = md
		anchors = append(anchors, md)
	}
	if sections["examples"] != "" {
		md, anchors = cleanupMarkDown(sections["examples"])
		cmd.Example = md
		anchors = append(anchors, md)
	}
	if len(anchors) > 0 {
		if cmd.Annotations == nil {
			cmd.Annotations = make(map[string]string)
		}
		cmd.Annotations["anchors"] = strings.Join(anchors, ",")
	}
}

type byName []*cobra.Command

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }
