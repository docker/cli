// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.25

package hooks

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

const maxMessages = 10

func ParseTemplate(hookTemplate string, cmd *cobra.Command) ([]string, error) {
	out := hookTemplate
	if strings.Contains(hookTemplate, "{{") {
		// Message may be a template.
		msgContext := commandInfo{cmd: cmd}

		tmpl, err := template.New("").Funcs(template.FuncMap{
			"command":   msgContext.command,
			"flagValue": msgContext.flagValue,
			"argValue":  msgContext.argValue,

			// kept for backward-compatibility with old templates.
			"flag": func(_ any, flagName string) (string, error) { return msgContext.flagValue(flagName) },
			"arg":  func(_ any, i int) (string, error) { return msgContext.argValue(i) },
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
	if n := strings.Count(out, "\n"); n > maxMessages {
		return nil, fmt.Errorf("hook template contains too many messages (%d): maximum is %d", n, maxMessages)
	}
	return strings.SplitN(out, "\n", maxMessages), nil
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
	return c.command()
}

// command returns the name of the (sub)command for which the hook was invoked.
func (c commandInfo) command() string {
	if c.cmd == nil {
		return ""
	}
	return c.cmd.Name()
}

// flagValue returns the value that was set for the given flag when the hook was invoked.
func (c commandInfo) flagValue(flagName string) (string, error) {
	if c.cmd == nil {
		return "", fmt.Errorf("%w: flagValue: cmd is nil", ErrHookTemplateParse)
	}
	f := c.cmd.Flag(flagName)
	if f == nil {
		return "", fmt.Errorf("%w: flagValue: no flags found", ErrHookTemplateParse)
	}
	return f.Value.String(), nil
}

// argValue returns the value of the nth argument.
func (c commandInfo) argValue(n int) (string, error) {
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
