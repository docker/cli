package stack

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/idresolver"
	"github.com/docker/cli/cli/command/stack/legacy/kubernetes"
	"github.com/docker/cli/cli/command/stack/legacy/swarm"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/task"
	cliopts "github.com/docker/cli/opts"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/docker/stacks/pkg/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newPsCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	opts := options.PS{Filter: cliopts.NewFilterOpt()}

	cmd := &cobra.Command{
		Use:   "ps [OPTIONS] STACK",
		Short: "List the tasks in the stack",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespace = args[0]
			if err := validateStackName(opts.Namespace); err != nil {
				return err
			}
			return RunPs(dockerCli, cmd.Flags(), common.Orchestrator(), opts)
		},
	}
	flags := cmd.Flags()
	flags.BoolVar(&opts.NoTrunc, "no-trunc", false, "Do not truncate output")
	flags.BoolVar(&opts.NoResolve, "no-resolve", false, "Do not map IDs to Names")
	flags.VarP(&opts.Filter, "filter", "f", "Filter output based on conditions provided")
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Only display task IDs")
	flags.StringVar(&opts.Format, "format", "", "Pretty-print tasks using a Go template")
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}

// RunPs performs a stack ps against the specified orchestrator
func RunPs(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.PS) error {
	if hasServerSideStacks(dockerCli) {
		return RunServerSidePs(dockerCli, flags, commonOrchestrator, opts)
	}
	return runLegacyOrchestratedCommand(dockerCli, flags, commonOrchestrator,
		func() error { return swarm.RunPS(dockerCli, opts) },
		func(kli *kubernetes.KubeCli) error { return kubernetes.RunPS(kli, opts) })
}

func RunServerSidePs(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.PS) error {
	ctx := context.Background()
	dclient := dockerCli.Client()
	// TODO - pending https://github.com/docker/stacks/pull/38
	// stack, err := getStackByName(ctx, dockerCli, string(commonOrchestrator), opts.Namespace)
	_, err := getStackByName(ctx, dockerCli, string(commonOrchestrator), opts.Namespace)
	if err != nil {
		return err
	}
	// TODO - pending https://github.com/docker/stacks/pull/38
	//tasks, err := dclient.StackTaskList(ctx, stack.ID)
	tasks := types.StackTaskList{}
	if err != nil {
		return err
	}
	swarmTasks := convertTaskList(tasks)
	if len(swarmTasks) == 0 {
		return fmt.Errorf("nothing found in stack: %s", opts.Namespace)
	}
	format := opts.Format
	if len(format) == 0 {
		format = task.DefaultFormat(dockerCli.ConfigFile(), opts.Quiet)
	}
	return task.Print(ctx, dockerCli, swarmTasks, idresolver.New(dclient, opts.NoResolve), !opts.NoTrunc, opts.Quiet, format)
}

func convertTaskList(tasks types.StackTaskList) []swarmtypes.Task {
	ret := []swarmtypes.Task{}
	allTasks := tasks.CurrentTasks
	allTasks = append(allTasks, tasks.PastTasks...)
	for _, task := range allTasks {
		ret = append(ret, swarmtypes.Task{
			ID: task.ID,
			Annotations: swarmtypes.Annotations{
				Name: task.Name,
				// Labels
			},
			NodeID: task.NodeID,
			Status: swarmtypes.TaskStatus{
				// Timestamp
				State: swarmtypes.TaskState(task.CurrentState),
				// Message
				Err: task.Err,
				// ContainerStatus
				// PortStatus
			},
			DesiredState: swarmtypes.TaskState(task.DesiredState),
			Spec: swarmtypes.TaskSpec{
				ContainerSpec: &swarmtypes.ContainerSpec{
					Image: task.Image,
					// Many omitted
				},
				// PluginSpec
				// NetworkAttachmentSpec
				// Resources
				// RestartPolicy
				// Placement
				// Networks
				// LogDriver
				// ForceUpdate
				// Runtime
			},
			// Meta {
			//      Version
			//      CreatedAt
			//      UpdatedAt
			// }
			// ServiceID
			// Slot
			// NetworkAttachments
			// GenericResources
		})
	}
	return ret
}
