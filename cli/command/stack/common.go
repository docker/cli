package stack

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/spf13/pflag"
)

// validateStackName checks if the provided string is a valid stack name (namespace).
// It currently only does a rudimentary check if the string is empty, or consists
// of only whitespace and quoting characters.
func validateStackName(namespace string) error {
	v := strings.TrimFunc(namespace, quotesOrWhitespace)
	if v == "" {
		return fmt.Errorf("invalid stack name: %q", namespace)
	}
	return nil
}

func validateStackNames(namespaces []string) error {
	for _, ns := range namespaces {
		if err := validateStackName(ns); err != nil {
			return err
		}
	}
	return nil
}

func quotesOrWhitespace(r rune) bool {
	return unicode.IsSpace(r) || r == '"' || r == '\''
}

func runOrchestratedCommand(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, swarmCmd func() error, kubernetesCmd func(*kubernetes.KubeCli) error) error {
	switch {
	case commonOrchestrator.HasAll():
		return errUnsupportedAllOrchestrator
	case commonOrchestrator.HasKubernetes():
		kli, err := kubernetes.WrapCli(dockerCli, kubernetes.NewOptions(flags, commonOrchestrator))
		if err != nil {
			return err
		}
		return kubernetesCmd(kli)
	default:
		return swarmCmd()
	}
}
