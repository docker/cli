package main

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

func registerCompletionFuncForGlobalFlags(dockerCli *command.DockerCli, cmd *cobra.Command) {
	cmd.RegisterFlagCompletionFunc(
		"context",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			names, err := store.Names(dockerCli.ContextStore())
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
	)
	cmd.RegisterFlagCompletionFunc(
		"log-level",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			values := []string{"debug", "info", "warn", "error", "fatal"}
			return values, cobra.ShellCompDirectiveNoFileComp
		},
	)
}
