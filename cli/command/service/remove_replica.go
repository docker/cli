package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

func newRemoveReplicaCommand(dockerCli *command.DockerCli) *cobra.Command {

	cmd := &cobra.Command{
		Use:     "rm-replica TASKNAME [TASKNAME...]",
		Aliases: []string{"remove-replica"},
		Short:   "Remove replicas from one or multiple replicated services",
		Args:    replicaArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemoveReplica(dockerCli, args)
		},
	}
	cmd.Flags()

	return cmd
}

func replicaArgs(cmd *cobra.Command, args []string) error {
	if err := cli.RequiresMinArgs(1)(cmd, args); err != nil {
		return err
	}
	for _, arg := range args {
		if parts := strings.SplitN(arg, ".", 2); len(parts) != 2 {
			return errors.Errorf(
				"Invalid task name '%s'.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
				arg,
				cmd.CommandPath(),
				cmd.UseLine(),
				cmd.Short,
			)
		}
	}
	return nil
}

func runRemoveReplica(dockerCli *command.DockerCli, args []string) error {
	var errs []string
	for _, arg := range args {
		parts := strings.SplitN(arg, ".", 2)
		serviceID, slotStr := parts[0], parts[1]

		// validate input arg slot number
		_, err := strconv.ParseUint(slotStr, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid slot number %s: %v", arg, slotStr, err))
			continue
		}

		if err := runServiceRemoveReplica(dockerCli, serviceID, slotStr); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", arg, err))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Errorf(strings.Join(errs, "\n"))
}

func runServiceRemoveReplica(dockerCli *command.DockerCli, serviceID string, slot string) error {
	client := dockerCli.Client()
	ctx := context.Background()

	service, _, err := client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	serviceMode := &service.Spec.Mode
	if serviceMode.Replicated == nil {
		return errors.Errorf("rm-replica can only be used with replicated mode")
	}

	err = client.ServiceRemoveReplica(ctx, serviceID, slot)
	if err != nil {
		return err
	}

	replicas := serviceMode.Replicated.Replicas

	fmt.Fprintf(dockerCli.Out(), "%s scaled to %d\n", serviceID, *replicas-1)
	return nil
}
