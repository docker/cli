// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.25

package formatter

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/docker/cli/cli/command/formatter/tabwriter"
	"github.com/docker/cli/templates"
)

// Format keys used to specify certain kinds of output formats
const (
	TableFormatKey  = "table"
	RawFormatKey    = "raw"
	PrettyFormatKey = "pretty"
	JSONFormatKey   = "json"

	DefaultQuietFormat = "{{.ID}}"
	JSONFormat         = "{{json .}}"
)

// Format is the format string rendered using the Context
type Format string

// IsTable returns true if the format is a table-type format
func (f Format) IsTable() bool {
	return strings.HasPrefix(string(f), TableFormatKey)
}

// IsJSON returns true if the format is the JSON format
func (f Format) IsJSON() bool {
	return string(f) == JSONFormatKey
}

// Contains returns true if the format contains the substring
func (f Format) Contains(sub string) bool {
	return strings.Contains(string(f), sub)
}

// templateString pre-processes the format and returns it as a string
// for templating.
func (f Format) templateString() string {
	out := string(f)
	switch out {
	case TableFormatKey:
		// A bare "--format table" should already be handled before we
		// hit this; a literal "table" here means a custom "table" format
		// without template.
		return ""
	case JSONFormatKey:
		// "--format json" only; not JSON formats ("--format '{{json .Field}}'").
		return JSONFormat
	}

	// "--format 'table {{.Field}}\t{{.Field}}'" -> "{{.Field}}\t{{.Field}}"
	if after, isTable := strings.CutPrefix(out, TableFormatKey); isTable {
		out = after
	}

	out = strings.Trim(out, " ") // trim spaces, but preserve other whitespace.
	out = strings.NewReplacer(`\t`, "\t", `\n`, "\n").Replace(out)
	return out
}

// Context contains information required by the formatter to print the output as desired.
type Context struct {
	// Output is the output stream to which the formatted string is written.
	Output io.Writer
	// Format is used to choose raw, table or custom format for the output.
	Format Format
	// Trunc when set to true will truncate the output of certain fields such as Container ID.
	Trunc bool

	// internal element
	header any
	buffer *bytes.Buffer
}

func (c *Context) parseFormat() (*template.Template, error) {
	tmpl, err := templates.Parse(c.Format.templateString())
	if err != nil {
		return nil, fmt.Errorf("template parsing error: %w", err)
	}
	return tmpl, nil
}

func (c *Context) postFormat(tmpl *template.Template, subContext SubContext) {
	out := c.Output
	if out == nil {
		out = io.Discard
	}
	if !c.Format.IsTable() {
		_, _ = c.buffer.WriteTo(out)
		return
	}

	// Write column-headers and rows to the tab-writer buffer, then flush the output.
	tw := tabwriter.NewWriter(out, 10, 1, 3, ' ', 0)
	_ = tmpl.Funcs(templates.HeaderFunctions).Execute(tw, subContext.FullHeader())
	_, _ = tw.Write([]byte{'\n'})
	_, _ = c.buffer.WriteTo(tw)
	_ = tw.Flush()
}

func (c *Context) contextFormat(tmpl *template.Template, subContext SubContext) error {
	if err := tmpl.Execute(c.buffer, subContext); err != nil {
		return fmt.Errorf("template parsing error: %w", err)
	}
	if c.Format.IsTable() && c.header != nil {
		c.header = subContext.FullHeader()
	}
	c.buffer.WriteString("\n")
	return nil
}

// SubFormat is a function type accepted by Write()
type SubFormat func(func(SubContext) error) error

// Write the template to the buffer using this Context
func (c *Context) Write(sub SubContext, f SubFormat) error {
	c.buffer = &bytes.Buffer{}
	tmpl, err := c.parseFormat()
	if err != nil {
		return err
	}

	subFormat := func(subContext SubContext) error {
		return c.contextFormat(tmpl, subContext)
	}
	if err := f(subFormat); err != nil {
		return err
	}

	c.postFormat(tmpl, sub)
	return nil
}
