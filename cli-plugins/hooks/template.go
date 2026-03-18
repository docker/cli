package hooks

import (
	"bytes"
	"errors"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

func ParseTemplate(hookTemplate string, cmd *cobra.Command) ([]string, error) {
	out := hookTemplate
	if strings.Contains(hookTemplate, "{{") {
		// Message may be a template.
		tmpl := template.New("").Funcs(commandFunctions)
		tmpl, err := tmpl.Parse(hookTemplate)
		if err != nil {
			return nil, err
		}
		var b bytes.Buffer
		err = tmpl.Execute(&b, cmd)
		if err != nil {
			return nil, err
		}
		out = b.String()
	}
	return strings.Split(out, "\n"), nil
}

var ErrHookTemplateParse = errors.New("failed to parse hook template")

var commandFunctions = template.FuncMap{
	"flag": getFlagValue,
	"arg":  getArgValue,
}

func getFlagValue(cmd *cobra.Command, flag string) (string, error) {
	cmdFlag := cmd.Flag(flag)
	if cmdFlag == nil {
		return "", ErrHookTemplateParse
	}
	return cmdFlag.Value.String(), nil
}

func getArgValue(cmd *cobra.Command, i int) (string, error) {
	flags := cmd.Flags()
	if flags == nil {
		return "", ErrHookTemplateParse
	}
	return flags.Arg(i), nil
}
