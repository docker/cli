package stack

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var errUnsupportedAllOrchestrator = fmt.Errorf(`no orchestrator specified: use either "kubernetes" or "swarm"`)

type commonOptions struct {
	orchestrator command.Orchestrator
}

// NewStackCommand returns a cobra command for `stack` subcommands
func NewStackCommand(dockerCli command.Cli) *cobra.Command {
	var opts commonOptions
	cmd := &cobra.Command{
		Use:   "stack [OPTIONS]",
		Short: "Manage Docker stacks",
		Args:  cli.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			orchestrator, err := getOrchestrator(dockerCli.ConfigFile(), cmd)
			if err != nil {
				return err
			}
			opts.orchestrator = orchestrator
			hideOrchestrationFlags(cmd, orchestrator)
			return checkSupportedFlag(cmd, orchestrator)
		},

		RunE: command.ShowHelp(dockerCli.Err()),
		Annotations: map[string]string{
			"version": "1.25",
		},
	}
	defaultHelpFunc := cmd.HelpFunc()
	cmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		hideOrchestrationFlags(cmd, opts.orchestrator)
		defaultHelpFunc(cmd, args)
	})
	cmd.AddCommand(
		newDeployCommand(dockerCli, &opts),
		newListCommand(dockerCli, &opts),
		newPsCommand(dockerCli, &opts),
		newRemoveCommand(dockerCli, &opts),
		newServicesCommand(dockerCli, &opts),
	)
	flags := cmd.PersistentFlags()
	flags.String("kubeconfig", "", "Kubernetes config file")
	flags.SetAnnotation("kubeconfig", "kubernetes", nil)
	flags.String("orchestrator", "", "Orchestrator to use (swarm|kubernetes|all)")
	return cmd
}

// NewTopLevelDeployCommand returns a command for `docker deploy`
func NewTopLevelDeployCommand(dockerCli command.Cli) *cobra.Command {
	cmd := newDeployCommand(dockerCli, nil)
	// Remove the aliases at the top level
	cmd.Aliases = []string{}
	cmd.Annotations = map[string]string{
		"experimental": "",
		"version":      "1.25",
	}
	return cmd
}

func getOrchestrator(config *configfile.ConfigFile, cmd *cobra.Command) (command.Orchestrator, error) {
	var orchestratorFlag string
	if o, err := cmd.Flags().GetString("orchestrator"); err == nil {
		orchestratorFlag = o
	}
	return command.GetStackOrchestrator(orchestratorFlag, config.StackOrchestrator)
}

func hideOrchestrationFlags(cmd *cobra.Command, orchestrator command.Orchestrator) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if _, ok := f.Annotations["kubernetes"]; ok && !orchestrator.HasKubernetes() {
			f.Hidden = true
		}
		if _, ok := f.Annotations["swarm"]; ok && !orchestrator.HasSwarm() {
			f.Hidden = true
		}
	})
	for _, subcmd := range cmd.Commands() {
		hideOrchestrationFlags(subcmd, orchestrator)
	}
}

func checkSupportedFlag(cmd *cobra.Command, orchestrator command.Orchestrator) error {
	errs := []string{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			return
		}
		if _, ok := f.Annotations["kubernetes"]; ok && !orchestrator.HasKubernetes() {
			errs = append(errs, fmt.Sprintf(`"--%s" is only supported on a Docker cli with kubernetes features enabled`, f.Name))
		}
		if _, ok := f.Annotations["swarm"]; ok && !orchestrator.HasSwarm() {
			errs = append(errs, fmt.Sprintf(`"--%s" is only supported on a Docker cli with swarm features enabled`, f.Name))
		}
	})
	for _, subcmd := range cmd.Commands() {
		if err := checkSupportedFlag(subcmd, orchestrator); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}
