package hooks

import (
	"fmt"
	"strconv"
)

const (
	hookTemplateCommandName = `{{.Name}}`
	hookTemplateFlagValue   = `{{flag . "%s"}}`
	hookTemplateArg         = `{{arg . %s}}`
)

// TemplateReplaceSubcommandName returns a hook template string
// that will be replaced by the CLI subcommand being executed
//
// Example:
//
// "you ran the subcommand: " + TemplateReplaceSubcommandName()
//
// when being executed after the command:
// `docker run --name "my-container" alpine`
// will result in the message:
// `you ran the subcommand: run`
func TemplateReplaceSubcommandName() string {
	return hookTemplateCommandName
}

// TemplateReplaceFlagValue returns a hook template string
// that will be replaced by the flags value.
//
// Example:
//
// "you ran a container named: " + TemplateReplaceFlagValue("name")
//
// when being executed after the command:
// `docker run --name "my-container" alpine`
// will result in the message:
// `you ran a container named: my-container`
func TemplateReplaceFlagValue(flag string) string {
	return fmt.Sprintf(hookTemplateFlagValue, flag)
}

// TemplateReplaceArg takes an index i and returns a hook
// template string that the CLI will replace the template with
// the ith argument, after processing the passed flags.
//
// Example:
//
// "run this image with `docker run " + TemplateReplaceArg(0) + "`"
//
// when being executed after the command:
// `docker pull alpine`
// will result in the message:
// "Run this image with `docker run alpine`"
func TemplateReplaceArg(i int) string {
	return fmt.Sprintf(hookTemplateArg, strconv.Itoa(i))
}
