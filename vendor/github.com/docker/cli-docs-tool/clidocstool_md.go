// Copyright 2021 cli-docs-tool authors
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
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/docker/cli-docs-tool/annotation"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	nlRegexp  = regexp.MustCompile(`\r?\n`)
	adjustSep = regexp.MustCompile(`\|:---(\s+)`)
)

// GenMarkdownTree will generate a markdown page for this command and all
// descendants in the directory given.
func (c *Client) GenMarkdownTree(cmd *cobra.Command) error {
	for _, sc := range cmd.Commands() {
		if err := c.GenMarkdownTree(sc); err != nil {
			return err
		}
	}

	// always disable the addition of [flags] to the usage
	cmd.DisableFlagsInUseLine = true

	// Skip the root command altogether, to prevent generating a useless
	// md file for plugins.
	if c.plugin && !cmd.HasParent() {
		return nil
	}

	// Skip hidden command
	if cmd.Hidden {
		log.Printf("INFO: Skipping Markdown for %q (hidden command)", cmd.CommandPath())
		return nil
	}

	log.Printf("INFO: Generating Markdown for %q", cmd.CommandPath())
	mdFile := mdFilename(cmd)
	sourcePath := filepath.Join(c.source, mdFile)
	targetPath := filepath.Join(c.target, mdFile)

	// check recursively to handle inherited annotations
	for curr := cmd; curr != nil; curr = curr.Parent() {
		if _, ok := cmd.Annotations[annotation.CodeDelimiter]; !ok {
			if cd, cok := curr.Annotations[annotation.CodeDelimiter]; cok {
				if cmd.Annotations == nil {
					cmd.Annotations = map[string]string{}
				}
				cmd.Annotations[annotation.CodeDelimiter] = cd
			}
		}
	}

	if !fileExists(sourcePath) {
		var icBuf bytes.Buffer
		icTpl, err := template.New("ic").Option("missingkey=error").Parse(`# {{ .Command }}

<!---MARKER_GEN_START-->
<!---MARKER_GEN_END-->

`)
		if err != nil {
			return err
		}
		if err = icTpl.Execute(&icBuf, struct {
			Command string
		}{
			Command: cmd.CommandPath(),
		}); err != nil {
			return err
		}
		if err = os.WriteFile(targetPath, icBuf.Bytes(), 0o644); err != nil {
			return err
		}
	} else if err := copyFile(sourcePath, targetPath); err != nil {
		return err
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		return err
	}

	cs := string(content)

	start := strings.Index(cs, "<!---MARKER_GEN_START-->")
	end := strings.Index(cs, "<!---MARKER_GEN_END-->")

	if start == -1 {
		return fmt.Errorf("no start marker in %s", mdFile)
	}
	if end == -1 {
		return fmt.Errorf("no end marker in %s", mdFile)
	}

	out, err := mdCmdOutput(cmd, cs)
	if err != nil {
		return err
	}
	cont := cs[:start] + "<!---MARKER_GEN_START-->" + "\n" + out + "\n" + cs[end:]

	fi, err := os.Stat(targetPath)
	if err != nil {
		return err
	}
	if err = os.WriteFile(targetPath, []byte(cont), fi.Mode()); err != nil {
		return fmt.Errorf("failed to write %s: %w", targetPath, err)
	}

	return nil
}

func mdFilename(cmd *cobra.Command) string {
	name := cmd.CommandPath()
	if i := strings.Index(name, " "); i >= 0 {
		name = name[i+1:]
	}
	return strings.ReplaceAll(name, " ", "_") + ".md"
}

func mdMakeLink(txt, link string, f *pflag.Flag, isAnchor bool) string {
	link = "#" + link
	annotations, ok := f.Annotations[annotation.ExternalURL]
	if ok && len(annotations) > 0 {
		link = annotations[0]
	} else {
		if !isAnchor {
			return txt
		}
	}

	return "[" + txt + "](" + link + ")"
}

type mdTable struct {
	out       *strings.Builder
	tabWriter *tabwriter.Writer
}

func newMdTable(headers ...string) *mdTable {
	w := &strings.Builder{}
	t := &mdTable{
		out: w,
		// Using tabwriter.Debug, which uses "|" as separator instead of tabs,
		// which is what we want. It's a bit of a hack, but does the job :)
		tabWriter: tabwriter.NewWriter(w, 5, 5, 1, ' ', tabwriter.Debug),
	}
	t.addHeader(headers...)
	return t
}

func (t *mdTable) addHeader(cols ...string) {
	t.AddRow(cols...)
	_, _ = t.tabWriter.Write([]byte("|" + strings.Repeat(":---\t", len(cols)) + "\n"))
}

func (t *mdTable) AddRow(cols ...string) {
	for i := range cols {
		cols[i] = mdEscapePipe(cols[i])
	}
	_, _ = t.tabWriter.Write([]byte("| " + strings.Join(cols, "\t ") + "\t\n"))
}

func (t *mdTable) String() string {
	_ = t.tabWriter.Flush()
	return adjustSep.ReplaceAllStringFunc(t.out.String()+"\n", func(in string) string {
		return strings.ReplaceAll(in, " ", "-")
	})
}

func mdCmdOutput(cmd *cobra.Command, old string) (string, error) {
	b := &strings.Builder{}

	desc := cmd.Short
	if cmd.Long != "" {
		desc = cmd.Long
	}
	if desc != "" {
		b.WriteString(desc + "\n\n")
	}

	if aliases := getAliases(cmd); len(aliases) != 0 {
		b.WriteString("### Aliases\n\n")
		b.WriteString("`" + strings.Join(aliases, "`, `") + "`")
		b.WriteString("\n\n")
	}

	if len(cmd.Commands()) != 0 {
		b.WriteString("### Subcommands\n\n")
		table := newMdTable("Name", "Description")
		for _, c := range cmd.Commands() {
			if c.Hidden {
				continue
			}
			table.AddRow(fmt.Sprintf("[`%s`](%s)", c.Name(), mdFilename(c)), c.Short)
		}
		b.WriteString(table.String() + "\n")
	}

	// add inherited flags before checking for flags availability
	cmd.Flags().AddFlagSet(cmd.InheritedFlags())

	if cmd.Flags().HasAvailableFlags() {
		b.WriteString("### Options\n\n")
		table := newMdTable("Name", "Type", "Default", "Description")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			isLink := strings.Contains(old, "<a name=\""+f.Name+"\"></a>")
			var name string
			if f.Shorthand != "" {
				name = mdMakeLink("`-"+f.Shorthand+"`", f.Name, f, isLink)
				name += ", "
			}
			name += mdMakeLink("`--"+f.Name+"`", f.Name, f, isLink)

			ftype := "`" + f.Value.Type() + "`"

			var defval string
			if v, ok := f.Annotations[annotation.DefaultValue]; ok && len(v) > 0 {
				defval = v[0]
				if cd, ok := f.Annotations[annotation.CodeDelimiter]; ok {
					defval = strings.ReplaceAll(defval, cd[0], "`")
				} else if cd, ok := cmd.Annotations[annotation.CodeDelimiter]; ok {
					defval = strings.ReplaceAll(defval, cd, "`")
				}
			} else if f.DefValue != "" && ((f.Value.Type() != "bool" && f.DefValue != "true") || (f.Value.Type() == "bool" && f.DefValue == "true")) && f.DefValue != "[]" {
				defval = "`" + f.DefValue + "`"
			}

			usage := f.Usage
			if cd, ok := f.Annotations[annotation.CodeDelimiter]; ok {
				usage = strings.ReplaceAll(usage, cd[0], "`")
			} else if cd, ok := cmd.Annotations[annotation.CodeDelimiter]; ok {
				usage = strings.ReplaceAll(usage, cd, "`")
			}
			table.AddRow(name, ftype, defval, mdReplaceNewline(usage))
		})
		b.WriteString(table.String())
	}

	return b.String(), nil
}

func mdEscapePipe(s string) string {
	return strings.ReplaceAll(s, `|`, `\|`)
}

func mdReplaceNewline(s string) string {
	return nlRegexp.ReplaceAllString(s, "<br>")
}
