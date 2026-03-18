package hooks

import (
	"fmt"
)

const (
	hookTemplateCommandName = `{{.Name}}`
	hookTemplateFlagValue   = `{{.FlagValue %q}}`
	hookTemplateArg         = `{{.Arg %d}}`
)

// TemplateReplaceSubcommandName returns a hook template string
// that will be replaced by the CLI subcommand being executed
//
// Example:
//
//	Response{
//		Type:     NextSteps,
//		Template: "you ran the subcommand: " + TemplateReplaceSubcommandName(),
//	}
//
// When being executed after the command:
//
//	docker run --name "my-container" alpine
//
// It results in the message:
//
//	you ran the subcommand: run
func TemplateReplaceSubcommandName() string {
	return hookTemplateCommandName
}

// TemplateReplaceFlagValue returns a hook template string that will be
// replaced with the flags value when printed by the CLI.
//
// Example:
//
//	Response{
//		Type:     NextSteps,
//		Template: "you ran a container named: " + TemplateReplaceFlagValue("name"),
//	}
//
// when executed after the command:
//
//	docker run --name "my-container" alpine
//
// it results in the message:
//
//	you ran a container named: my-container
func TemplateReplaceFlagValue(flag string) string {
	return fmt.Sprintf(hookTemplateFlagValue, flag)
}

// TemplateReplaceArg takes an index i and returns a hook
// template string that the CLI will replace the template with
// the ith argument after processing the passed flags.
//
// Example:
//
//	Response{
//		Type:     NextSteps,
//		Template: "run this image with `docker run " + TemplateReplaceArg(0) + "`",
//	}
//
// when being executed after the command:
//
//	docker pull alpine
//
// It results in the message:
//
//	Run this image with `docker run alpine`
func TemplateReplaceArg(i int) string {
	return fmt.Sprintf(hookTemplateArg, i)
}
