package kubernetes

import (
	"fmt"
	"sort"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/command/task"
	"github.com/docker/cli/opts"
	"github.com/docker/docker/api/types/swarm"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/kubernetes/pkg/api"
)

type psOptions struct {
	filter    opts.FilterOpt
	noTrunc   bool
	namespace string
	noResolve bool
	quiet     bool
	format    string
}

func newPsCommand(dockerCli command.Cli, kubeCli *kubeCli) *cobra.Command {
	options := psOptions{filter: opts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS] STACK",
		Short: "List the tasks in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.namespace = args[0]
			return runPS(dockerCli, kubeCli, options)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&options.noTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&options.noResolve, "no-resolve", false, "Do not map IDs to Names")
	flags.VarP(&options.filter, "filter", "f", "Filter output based on conditions provided")
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only display task IDs")
	flags.StringVar(&options.format, "format", "", "Pretty-print tasks using a Go template")

	return cmd
}

func runPS(dockerCli command.Cli, kubeCli *kubeCli, options psOptions) error {
	namespace := options.namespace

	// Initialize clients
	client, err := kubeCli.ComposeClient()
	if err != nil {
		return err
	}
	stacks, err := kubeCli.Stacks()
	if err != nil {
		return err
	}
	podsClient := client.Pods()

	// Fetch pods
	if _, err := stacks.Get(namespace, metav1.GetOptions{}); err != nil {
		return fmt.Errorf("nothing found in stack: %s", namespace)
	}

	pods, err := fetchPods(namespace, podsClient)
	if err != nil {
		return err
	}

	if len(pods) == 0 {
		return fmt.Errorf("nothing found in stack: %s", namespace)
	}

	format := options.format
	if len(format) == 0 {
		format = task.DefaultFormat(dockerCli.ConfigFile(), options.quiet)
	}
	nodeResolver := makeNodeResolver(options.noResolve, client.Nodes())

	tasks := make([]swarm.Task, len(pods))
	for i, pod := range pods {
		tasks[i] = podToTask(pod)
	}
	return print(dockerCli, tasks, pods, nodeResolver, !options.noTrunc, options.quiet, format)
}

type idResolver func(name string) (string, error)

func print(dockerCli command.Cli, tasks []swarm.Task, pods []apiv1.Pod, nodeResolver idResolver, trunc, quiet bool, format string) error {
	sort.Stable(tasksBySlot(tasks))

	names := map[string]string{}
	nodes := map[string]string{}

	tasksCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.NewTaskFormat(format, quiet),
		Trunc:  trunc,
	}

	for i, task := range tasks {
		nodeValue, err := nodeResolver(pods[i].Spec.NodeName)
		if err != nil {
			return err
		}

		names[task.ID] = pods[i].Name
		nodes[task.ID] = nodeValue
	}

	return formatter.TaskWrite(tasksCtx, tasks, names, nodes)
}

func makeNodeResolver(noResolve bool, nodes corev1.NodeInterface) func(string) (string, error) {
	// Here we have a name and we need to resolve its identifier. To mimic swarm behavior
	// we need to resolve the id when noresolve is set, otherwise we return the name.
	if noResolve {
		return func(name string) (string, error) {
			n, err := nodes.List(metav1.ListOptions{
				FieldSelector: fields.OneTermEqualSelector(api.ObjectNameField, name).String(),
			})
			if err != nil {
				return "", err
			}
			if len(n.Items) != 1 {
				return "", fmt.Errorf("could not find node '%s'", name)
			}
			return string(n.Items[0].UID), nil
		}
	}
	return func(name string) (string, error) { return name, nil }
}
