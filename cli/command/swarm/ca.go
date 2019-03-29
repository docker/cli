package swarm

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/swarm/progress"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/jsonmessage"
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
			return runCA(dockerCli, cmd.Flags(), opts)
		},
		Annotations: map[string]string{"version": "1.30"},
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

func runCA(dockerCli command.Cli, flags *pflag.FlagSet, opts caOptions) error {
	client := dockerCli.Client()
	ctx := context.Background()

	swarmInspect, err := client.SwarmInspect(ctx)
	if err != nil {
		return err
	}

	if !opts.rotate {
		for _, f := range []string{flagCACert, flagCAKey, flagCertExpiry, flagExternalCA} {
			if flags.Changed(f) {
				return fmt.Errorf("`--%s` flag requires the `--rotate` flag to update the CA", f)
			}
		}
		return displayTrustRoot(dockerCli.Out(), swarmInspect)
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
	if err := client.SwarmUpdate(ctx, swarmInspect.Version, swarmInspect.Spec, swarm.UpdateFlags{}); err != nil {
		return err
	}

	if opts.detach {
		return nil
	}
	return attach(ctx, dockerCli, opts)
}

func updateSwarmSpec(spec *swarm.Spec, flags *pflag.FlagSet, opts caOptions) {
	caCert := opts.rootCACert.Contents()
	caKey := opts.rootCAKey.Contents()
	opts.mergeSwarmSpecCAFlags(spec, flags, caCert)

	spec.CAConfig.SigningCACert = caCert
	spec.CAConfig.SigningCAKey = caKey

	if caKey == "" && caCert == "" {
		spec.CAConfig.ForceRotate++
	}
}

func attach(ctx context.Context, dockerCli command.Cli, opts caOptions) error {
	client := dockerCli.Client()
	errChan := make(chan error, 1)
	pipeReader, pipeWriter := io.Pipe()

	go func() {
		errChan <- progress.RootRotationProgress(ctx, client, pipeWriter)
	}()

	if opts.quiet {
		go io.Copy(ioutil.Discard, pipeReader)
		return <-errChan
	}

	err := jsonmessage.DisplayJSONMessagesToStream(pipeReader, dockerCli.Out(), nil)
	if err == nil {
		err = <-errChan
	}
	if err != nil {
		return err
	}

	swarmInspect, err := client.SwarmInspect(ctx)
	if err != nil {
		return err
	}
	return displayTrustRoot(dockerCli.Out(), swarmInspect)
}

func displayTrustRoot(out io.Writer, info swarm.Swarm) error {
	if info.ClusterInfo.TLSInfo.TrustRoot == "" {
		return errors.New("No CA information available")
	}
	fmt.Fprintln(out, strings.TrimSpace(info.ClusterInfo.TLSInfo.TrustRoot))
	return nil
}
