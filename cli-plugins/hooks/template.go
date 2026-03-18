package hooks

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

func ParseTemplate(hookTemplate string, cmd *cobra.Command) ([]string, error) {
	out := hookTemplate
	if strings.Contains(hookTemplate, "{{") {
		// Message may be a template.
		msgContext := commandInfo{cmd: cmd}

		tmpl, err := template.New("").Funcs(template.FuncMap{
			// kept for backward-compatibility with old templates.
			"flag": func(_ any, flagName string) (string, error) { return msgContext.FlagValue(flagName) },
			"arg":  func(_ any, i int) (string, error) { return msgContext.Arg(i) },
		}).Parse(hookTemplate)
		if err != nil {
			return nil, err
		}
		var b bytes.Buffer
		err = tmpl.Execute(&b, msgContext)
		if err != nil {
			return nil, err
		}
		out = b.String()
	}
	return strings.Split(out, "\n"), nil
}

var ErrHookTemplateParse = errors.New("failed to parse hook template")

// commandInfo provides info about the command for which the hook was invoked.
// It is used for templated hook-messages.
type commandInfo struct {
	cmd *cobra.Command
}

// Name returns the name of the (sub)command for which the hook was invoked.
//
// It's used for backward-compatibility with old templates.
func (c commandInfo) Name() string {
	if c.cmd == nil {
		return ""
	}
	return c.cmd.Name()
}

// FlagValue returns the value that was set for the given flag when the hook was invoked.
func (c commandInfo) FlagValue(flagName string) (string, error) {
	if c.cmd == nil {
		return "", fmt.Errorf("%w: flagValue: cmd is nil", ErrHookTemplateParse)
	}
	f := c.cmd.Flag(flagName)
	if f == nil {
		return "", fmt.Errorf("%w: flagValue: no flags found", ErrHookTemplateParse)
	}
	return f.Value.String(), nil
}

// Arg returns the value of the nth argument.
func (c commandInfo) Arg(n int) (string, error) {
	if c.cmd == nil {
		return "", fmt.Errorf("%w: arg: cmd is nil", ErrHookTemplateParse)
	}
	flags := c.cmd.Flags()
	v := flags.Arg(n)
	if v == "" && n >= flags.NArg() {
		return "", fmt.Errorf("%w: arg: %dth argument not set", ErrHookTemplateParse, n)
	}
	return v, nil
}
