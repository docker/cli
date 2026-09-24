package network

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/opts"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

type editOptions struct {
	labelsToAdd opts.ListOpts
	labelsToRm  []string
}

func newEditCommand(dockerCLI command.Cli) *cobra.Command {
	options := editOptions{
		labelsToAdd: opts.NewListOpts(opts.ValidateLabel),
	}

	cmd := &cobra.Command{
		Use:   "edit [OPTIONS] NETWORK",
		Short: "Edit a network",
		Long: `Edit the labels of a network.

Because the Docker Engine API does not support in-place network updates, this
command recreates the network with the same configuration and updated labels.
The network must have no active endpoints before editing; use
'docker network disconnect' to disconnect any containers first.`,
		Args: cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEdit(cmd.Context(), dockerCLI.Client(), dockerCLI.Out(), args[0], options)
		},
		ValidArgsFunction:     completion.NetworkNames(dockerCLI),
		DisableFlagsInUseLine: true,
	}

	flags := cmd.Flags()
	flags.Var(&options.labelsToAdd, "label-add", "Add or update a label (format: `key=value`)")
	flags.StringSliceVar(&options.labelsToRm, "label-rm", nil, "Remove a label by key")
	return cmd
}

func runEdit(ctx context.Context, apiClient client.NetworkAPIClient, output io.Writer, networkID string, options editOptions) error {
	if options.labelsToAdd.Len() == 0 && len(options.labelsToRm) == 0 {
		return fmt.Errorf("no changes requested; use --label-add or --label-rm to modify labels")
	}

	result, err := apiClient.NetworkInspect(ctx, networkID, client.NetworkInspectOptions{})
	if err != nil {
		return err
	}
	nw := result.Network

	if len(nw.Containers) > 0 {
		var names []string
		for _, ep := range nw.Containers {
			names = append(names, ep.Name)
		}
		return fmt.Errorf("network %s has active endpoints (%s); disconnect all containers before editing",
			nw.Name, strings.Join(names, ", "))
	}

	// Build updated labels from existing ones.
	labels := make(map[string]string, len(nw.Labels))
	for k, v := range nw.Labels {
		labels[k] = v
	}
	for _, l := range options.labelsToAdd.GetSlice() {
		k, v, _ := strings.Cut(l, "=")
		labels[k] = v
	}
	for _, k := range options.labelsToRm {
		delete(labels, k)
	}

	// NetworkRemove is called first; if it fails the original network is left intact.
	if _, err = apiClient.NetworkRemove(ctx, nw.ID, client.NetworkRemoveOptions{}); err != nil {
		return fmt.Errorf("NetworkRemove: %w", err)
	}

	// Preserve EnableIPv4/EnableIPv6 as pointers so that false values are
	// explicitly sent to the daemon (matching the original network's settings).
	enableIPv4 := nw.EnableIPv4
	enableIPv6 := nw.EnableIPv6

	resp, err := apiClient.NetworkCreate(ctx, nw.Name, client.NetworkCreateOptions{
		Driver:     nw.Driver,
		Options:    nw.Options,
		IPAM:       &nw.IPAM,
		Internal:   nw.Internal,
		EnableIPv4: &enableIPv4,
		EnableIPv6: &enableIPv6,
		Attachable: nw.Attachable,
		Ingress:    nw.Ingress,
		Scope:      nw.Scope,
		ConfigOnly: nw.ConfigOnly,
		ConfigFrom: nw.ConfigFrom.Network,
		Labels:     labels,
	})
	if err != nil {
		return fmt.Errorf("NetworkCreate: %w", err)
	}
	_, _ = fmt.Fprintln(output, resp.ID)
	return nil
}
