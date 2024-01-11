package main

import (
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
	err = cmd.RegisterFlagCompletionFunc(
		"log-level",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			values := []string{"debug", "info", "warn", "error", "fatal"}
			return values, cobra.ShellCompDirectiveNoFileComp
		},
	)
	if err != nil {
		return err
	}

	return nil
}
