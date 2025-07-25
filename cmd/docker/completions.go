package main

import (
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

type contextStoreProvider interface {
	ContextStore() store.Store
}

func completeContextNames(dockerCLI contextStoreProvider) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		names, err := store.Names(dockerCLI.ContextStore())
		if err != nil {
			cobra.CompErrorln("failed to get context names: " + err.Error())
			return nil, cobra.ShellCompDirectiveError
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

var logLevels = []string{"debug", "info", "warn", "error", "fatal", "panic"}

func completeLogLevels(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return cobra.FixedCompletions(logLevels, cobra.ShellCompDirectiveNoFileComp)(nil, nil, "")
}
