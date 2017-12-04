package kubernetes

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/kubernetes/labels"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type servicesOptions struct {
	quiet     bool
	format    string
	namespace string
}

func newServicesCommand(dockerCli command.Cli, kubeCli *kubeCli) *cobra.Command {
	var options servicesOptions

	cmd := &cobra.Command{
		Use:   "services [OPTIONS] STACK",
		Short: "List the services in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.namespace = args[0]
			return runServices(dockerCli, kubeCli, options)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display IDs")
	flags.StringVar(&options.format, "format", "", "Pretty-print services using a Go template")

	return cmd
}

func runServices(dockerCli command.Cli, kubeCli *kubeCli, options servicesOptions) error {
	// Initialize clients
	client, err := kubeCli.ComposeClient()
	if err != nil {
		return nil
	}
	stacks, err := kubeCli.Stacks()
	if err != nil {
		return err
	}
	replicas := client.ReplicaSets()

	if _, err := stacks.Get(options.namespace, metav1.GetOptions{}); err != nil {
		fmt.Fprintf(dockerCli.Err(), "Nothing found in stack: %s\n", options.namespace)
		return nil
	}

	replicasList, err := replicas.List(metav1.ListOptions{LabelSelector: labels.SelectorForStack(options.namespace)})
	if err != nil {
		return err
	}

	servicesList, err := client.Services().List(metav1.ListOptions{LabelSelector: labels.SelectorForStack(options.namespace)})
	if err != nil {
		return err
	}

	// Convert Replicas sets and kubernetes services to swam services and formatter informations
	services, info, err := replicasToServices(replicasList, servicesList)
	if err != nil {
		return err
	}

	if options.quiet {
		info = map[string]formatter.ServiceListInfo{}
	}

	format := options.format
	if len(format) == 0 {
		if len(dockerCli.ConfigFile().ServicesFormat) > 0 && !options.quiet {
			format = dockerCli.ConfigFile().ServicesFormat
		} else {
			format = formatter.TableFormatKey
		}
	}

	servicesCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewServiceListFormat(format, options.quiet),
	}
	return formatter.ServiceListWrite(servicesCtx, services, info)
}
