package swarm

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/swarm/progress"
	"github.com/docker/cli/internal/jsonstream"
	"github.com/moby/moby/api/types/swarm"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type caOptions struct {
	swarmCAOptions
	rootCACert PEMFile
	rootCAKey  PEMFile
	rotate     bool
	detach     bool
	quiet      bool
}

func newCACommand(dockerCli command.Cli) *cobra.Command {
	opts := caOptions{}

	cmd := &cobra.Command{
		Use:   "ca [OPTIONS]",
		Short: "Display and rotate the root CA",
		Args:  cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCA(cmd.Context(), dockerCli, cmd.Flags(), opts)
		},
		Annotations: map[string]string{
			"version": "1.30",
			"swarm":   "manager",
		},
		ValidArgsFunction: completion.NoComplete,
	}

	flags := cmd.Flags()
	addSwarmCAFlags(flags, &opts.swarmCAOptions)
	flags.BoolVar(&opts.rotate, flagRotate, false, "Rotate the swarm CA - if no certificate or key are provided, new ones will be generated")
	flags.Var(&opts.rootCACert, flagCACert, "Path to the PEM-formatted root CA certificate to use for the new cluster")
	flags.Var(&opts.rootCAKey, flagCAKey, "Path to the PEM-formatted root CA key to use for the new cluster")

	flags.BoolVarP(&opts.detach, "detach", "d", false, "Exit immediately instead of waiting for the root rotation to converge")
	flags.BoolVarP(&opts.quiet, "quiet", "q", false, "Suppress progress output")
	return cmd
}

func runCA(ctx context.Context, dockerCLI command.Cli, flags *pflag.FlagSet, opts caOptions) error {
	apiClient := dockerCLI.Client()

	swarmInspect, err := apiClient.SwarmInspect(ctx)
	if err != nil {
		return err
	}

	if !opts.rotate {
		for _, f := range []string{flagCACert, flagCAKey, flagCertExpiry, flagExternalCA} {
			if flags.Changed(f) {
				return fmt.Errorf("`--%s` flag requires the `--rotate` flag to update the CA", f)
			}
		}
		return displayTrustRoot(dockerCLI.Out(), swarmInspect)
	}

	if flags.Changed(flagExternalCA) && len(opts.externalCA.Value()) > 0 && !flags.Changed(flagCACert) {
		return fmt.Errorf(
			"rotating to an external CA requires the `--%s` flag to specify the external CA's cert - "+
				"to add an external CA with the current root CA certificate, use the `update` command instead", flagCACert)
	}

	if flags.Changed(flagCACert) && len(opts.externalCA.Value()) == 0 && !flags.Changed(flagCAKey) {
		return fmt.Errorf("the --%s flag requires that a --%s flag and/or --%s flag be provided as well",
			flagCACert, flagCAKey, flagExternalCA)
	}

	updateSwarmSpec(&swarmInspect.Spec, flags, opts)
	if err := apiClient.SwarmUpdate(ctx, swarmInspect.Version, swarmInspect.Spec, client.SwarmUpdateFlags{}); err != nil {
		return err
	}

	if opts.detach {
		return nil
	}
	return attach(ctx, dockerCLI, opts)
}

func updateSwarmSpec(spec *swarm.Spec, flags *pflag.FlagSet, opts caOptions) {
	caCert := opts.rootCACert.Contents()
	caKey := opts.rootCAKey.Contents()
	opts.mergeSwarmSpecCAFlags(spec, flags, &caCert)

	spec.CAConfig.SigningCACert = caCert
	spec.CAConfig.SigningCAKey = caKey

	if caKey == "" && caCert == "" {
		spec.CAConfig.ForceRotate++
	}
}

func attach(ctx context.Context, dockerCLI command.Cli, opts caOptions) error {
	apiClient := dockerCLI.Client()
	errChan := make(chan error, 1)
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		errChan <- progress.RootRotationProgress(ctx, apiClient, pipeWriter)
	}()

	if opts.quiet {
		go io.Copy(io.Discard, pipeReader)
		return <-errChan
	}

	err := jsonstream.Display(ctx, pipeReader, dockerCLI.Out())
	if err == nil {
		err = <-errChan
	}
	if err != nil {
		return err
	}

	swarmInspect, err := apiClient.SwarmInspect(ctx)
	if err != nil {
		return err
	}
	return displayTrustRoot(dockerCLI.Out(), swarmInspect)
}

func displayTrustRoot(out io.Writer, info swarm.Swarm) error {
	if info.ClusterInfo.TLSInfo.TrustRoot == "" {
		return errors.New("No CA information available")
	}
	_, _ = fmt.Fprintln(out, strings.TrimSpace(info.ClusterInfo.TLSInfo.TrustRoot))
	return nil
}
