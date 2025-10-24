package node

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var errNoRoleChange = errors.New("role was already set to the requested value")

func newUpdateCommand(dockerCLI command.Cli) *cobra.Command {
	options := newNodeOptions()

	cmd := &cobra.Command{
		Use:   "update [OPTIONS] NODE",
		Short: "Update a node",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), dockerCLI, cmd.Flags(), args[0])
		},
		ValidArgsFunction:     completeNodeNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.StringVar(&options.role, flagRole, "", `Role of the node ("worker", "manager")`)
	flags.StringVar(&options.availability, flagAvailability, "", `Availability of the node ("active", "pause", "drain")`)
	flags.Var(&options.annotations.labels, flagLabelAdd, `Add or update a node label ("key=value")`)
	labelKeys := opts.NewListOpts(nil)
	flags.Var(&labelKeys, flagLabelRemove, "Remove a node label if exists")

	_ = cmd.RegisterFlagCompletionFunc(flagRole, completion.FromList("worker", "manager"))
	_ = cmd.RegisterFlagCompletionFunc(flagAvailability, completion.FromList("active", "pause", "drain"))

	return cmd
}

func runUpdate(ctx context.Context, dockerCLI command.Cli, flags *pflag.FlagSet, nodeID string) error {
	return updateNodes(ctx, dockerCLI.Client(), []string{nodeID}, mergeNodeUpdate(flags), func(_ string) {
		_, _ = fmt.Fprintln(dockerCLI.Out(), nodeID)
	})
}

func updateNodes(ctx context.Context, apiClient client.NodeAPIClient, nodes []string, mergeNode func(node *swarm.Node) error, success func(nodeID string)) error {
	for _, nodeID := range nodes {
		res, err := apiClient.NodeInspect(ctx, nodeID, client.NodeInspectOptions{})
		if err != nil {
			return err
		}

		err = mergeNode(&res.Node)
		if err != nil {
			if errors.Is(err, errNoRoleChange) {
				continue
			}
			return err
		}
		_, err = apiClient.NodeUpdate(ctx, res.Node.ID, client.NodeUpdateOptions{
			Version: res.Node.Version,
			Spec:    res.Node.Spec,
		})
		if err != nil {
			return err
		}
		success(nodeID)
	}
	return nil
}

func mergeNodeUpdate(flags *pflag.FlagSet) func(*swarm.Node) error {
	return func(node *swarm.Node) error {
		spec := &node.Spec

		if flags.Changed(flagRole) {
			str, err := flags.GetString(flagRole)
			if err != nil {
				return err
			}
			spec.Role = swarm.NodeRole(str)
		}
		if flags.Changed(flagAvailability) {
			str, err := flags.GetString(flagAvailability)
			if err != nil {
				return err
			}
			spec.Availability = swarm.NodeAvailability(str)
		}
		if spec.Annotations.Labels == nil {
			spec.Annotations.Labels = make(map[string]string)
		}
		if flags.Changed(flagLabelAdd) {
			labels := flags.Lookup(flagLabelAdd).Value.(*opts.ListOpts).GetSlice()
			for k, v := range opts.ConvertKVStringsToMap(labels) {
				spec.Annotations.Labels[k] = v
			}
		}
		if flags.Changed(flagLabelRemove) {
			keys := flags.Lookup(flagLabelRemove).Value.(*opts.ListOpts).GetSlice()
			for _, k := range keys {
				// if a key doesn't exist, fail the command explicitly
				if _, exists := spec.Annotations.Labels[k]; !exists {
					return fmt.Errorf("key %s doesn't exist in node's labels", k)
				}
				delete(spec.Annotations.Labels, k)
			}
		}
		return nil
	}
}

const (
	flagRole         = "role"
	flagAvailability = "availability"
	flagLabelAdd     = "label-add"
	flagLabelRemove  = "label-rm"
)
