package main

import (
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

type contextStoreProvider interface {
	ContextStore() store.Store
}

func registerCompletionFuncForGlobalFlags(dockerCLI contextStoreProvider, cmd *cobra.Command) error {
	err := cmd.RegisterFlagCompletionFunc("context", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		names, _ := store.Names(dockerCLI.ContextStore())
		return names, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		return err
	}
	err = cmd.RegisterFlagCompletionFunc("log-level", completion.FromList("debug", "info", "warn", "error", "fatal"))
	if err != nil {
		return err
	}

	return nil
}
