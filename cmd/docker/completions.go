package main

import (
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

func registerCompletionFuncForGlobalFlags(contextStore store.Store, cmd *cobra.Command) error {
	err := cmd.RegisterFlagCompletionFunc(
		"context",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			names, err := store.Names(contextStore)
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
	)
	if err != nil {
		return err
	}
	err = cmd.RegisterFlagCompletionFunc("log-level", completion.FromList("debug", "info", "warn", "error", "fatal"))
	if err != nil {
		return err
	}

	return nil
}
