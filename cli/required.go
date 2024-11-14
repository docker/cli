package cli

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NoArgs validates args and returns an error if there are any args
func NoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}

	if cmd.HasSubCommands() {
		return errors.New("\n" + strings.TrimRight(cmd.UsageString(), "\n"))
	}

	return errors.Errorf(
		"%q accepts no arguments.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
		cmd.CommandPath(),
		cmd.CommandPath(),
		cmd.UseLine(),
		cmd.Short,
	)
}

// RequiresMinArgs returns an error if there is not at least min args
func RequiresMinArgs(minArgs int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) >= minArgs {
			return nil
		}
		return errors.Errorf(
			"%q requires at least %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			minArgs,
			pluralize("argument", minArgs),
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

// RequiresMaxArgs returns an error if there is not at most max args
func RequiresMaxArgs(maxArgs int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) <= maxArgs {
			return nil
		}
		return errors.Errorf(
			"%q requires at most %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			maxArgs,
			pluralize("argument", maxArgs),
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

// RequiresRangeArgs returns an error if there is not at least min args and at most max args
func RequiresRangeArgs(minArgs int, maxArgs int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) >= minArgs && len(args) <= maxArgs {
			return nil
		}
		return errors.Errorf(
			"%q requires at least %d and at most %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			minArgs,
			maxArgs,
			pluralize("argument", maxArgs),
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

// ExactArgs returns an error if there is not the exact number of args
func ExactArgs(number int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == number {
			return nil
		}
		return errors.Errorf(
			"%q requires exactly %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			number,
			pluralize("argument", number),
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

//nolint:unparam
func pluralize(word string, number int) string {
	if number == 1 {
		return word
	}
	return word + "s"
}
