package stack

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/legacy/kubernetes"
	legacyloader "github.com/docker/cli/cli/command/stack/legacy/loader"
	"github.com/docker/cli/cli/command/stack/legacy/swarm"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/stacks/pkg/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newDeployCommand(dockerCli command.Cli, common *commonOptions) *cobra.Command {
	var opts options.Deploy

	cmd := &cobra.Command{
		Use:     "deploy [OPTIONS] STACK",
		Aliases: []string{"up"},
		Short:   "Deploy a new stack or update an existing stack",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Namespace = args[0]
			if err := validateStackName(opts.Namespace); err != nil {
				return err
			}

			commonOrchestrator := command.OrchestratorSwarm // default for top-level deploy command
			if common != nil {
				commonOrchestrator = common.orchestrator
			}

			switch {
			case opts.Bundlefile == "" && len(opts.Composefiles) == 0:
				return errors.Errorf("Please specify either a bundle file (with --bundle-file) or a Compose file (with --compose-file).")
			case opts.Bundlefile != "" && len(opts.Composefiles) != 0:
				return errors.Errorf("You cannot specify both a bundle file and a Compose file.")
			case opts.Bundlefile != "":
				if commonOrchestrator != command.OrchestratorSwarm {
					return errors.Errorf("bundle files are not supported on another orchestrator than swarm.")
				}
				return swarm.DeployBundle(context.Background(), dockerCli, opts)
			}

			return RunDeploy(dockerCli, cmd.Flags(), common.Orchestrator(), opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Bundlefile, "bundle-file", "", "Path to a Distributed Application Bundle file")
	flags.SetAnnotation("bundle-file", "experimental", nil)
	flags.SetAnnotation("bundle-file", "swarm", nil)
	flags.StringSliceVarP(&opts.Composefiles, "compose-file", "c", []string{}, `Path to a Compose file, or "-" to read from stdin`)
	flags.SetAnnotation("compose-file", "version", []string{"1.25"})
	flags.BoolVar(&opts.SendRegistryAuth, "with-registry-auth", false, "Send registry authentication details to Swarm agents")
	flags.SetAnnotation("with-registry-auth", "swarm", nil)
	flags.BoolVar(&opts.Prune, "prune", false, "Prune services that are no longer referenced") // TODO - deprecate
	flags.SetAnnotation("prune", "version", []string{"1.27"})
	flags.SetAnnotation("prune", "swarm", nil)
	flags.StringVar(&opts.ResolveImage, "resolve-image", swarm.ResolveImageAlways,
		`Query the registry to resolve image digest and supported platforms ("`+swarm.ResolveImageAlways+`"|"`+swarm.ResolveImageChanged+`"|"`+swarm.ResolveImageNever+`")`) // TODO - deprecate
	flags.SetAnnotation("resolve-image", "version", []string{"1.30"})
	flags.SetAnnotation("resolve-image", "swarm", nil)
	kubernetes.AddNamespaceFlag(flags)
	return cmd
}

// RunDeploy performs a stack deploy against the specified orchestrator
func RunDeploy(dockerCli command.Cli, flags *pflag.FlagSet, commonOrchestrator command.Orchestrator, opts options.Deploy) error {
	if hasServerSideStacks(dockerCli) {
		ctx := context.Background()
		stackCreate, err := LoadComposefile(ctx, dockerCli, opts)
		if err != nil {
			return err
		}
		return runServerSideDeploy(ctx, dockerCli, stackCreate, commonOrchestrator, opts)
	}
	config, err := legacyloader.LoadComposefile(dockerCli, opts)
	if err != nil {
		return err
	}
	return runLegacyOrchestratedCommand(dockerCli, flags, commonOrchestrator,
		func() error { return swarm.RunDeploy(dockerCli, opts, config) },
		func(kli *kubernetes.KubeCli) error { return kubernetes.RunDeploy(kli, opts, config) })
}

func runServerSideDeploy(ctx context.Context, dockerCli command.Cli, stackCreate *types.StackCreate, commonOrchestrator command.Orchestrator, opts options.Deploy) error {
	dclient := dockerCli.Client()
	name := opts.Namespace

	var encodedAuth string
	var err error
	if opts.SendRegistryAuth {
		encodedAuth, err = getAuthHeaderForStack(ctx, dockerCli, stackCreate)
		if err != nil {
			return err
		}
	}

	// Check for existence first and update if found
	stack, err := getStackByName(ctx, dockerCli, string(commonOrchestrator), name)
	if err == nil {
		fmt.Fprintf(dockerCli.Out(), "Updating stack %s\n", name)
		updateOpts := types.StackUpdateOptions{
			EncodedRegistryAuth: encodedAuth,
		}
		return dclient.StackUpdate(ctx, stack.ID, stack.Version, stackCreate.Spec, updateOpts)
	}

	stackCreate.Orchestrator = types.OrchestratorChoice(commonOrchestrator)
	stackCreate.Metadata.Name = name

	createOpts := types.StackCreateOptions{
		EncodedRegistryAuth: encodedAuth,
	}
	resp, err := dclient.StackCreate(ctx, *stackCreate, createOpts)
	if err != nil {
		return err
	}

	// TODO Wait for the stack to stabilise - mimic legacy/kubernetes/deploy.go

	fmt.Fprintf(dockerCli.Out(), "Deployed Stack %s\n", resp.ID)

	return nil
}
