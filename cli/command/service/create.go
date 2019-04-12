package service

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	cliopts "github.com/docker/cli/opts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newCreateCommand(dockerCli command.Cli) *cobra.Command {
	opts := newServiceOptions()

	cmd := &cobra.Command{
		Use:   "create [OPTIONS] IMAGE [COMMAND] [ARG...]",
		Short: "Create a new service",
		Args:  cli.RequiresMinArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.image = args[0]
			if len(args) > 1 {
				opts.args = args[1:]
			}
			return runCreate(dockerCli, cmd.Flags(), opts)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.mode, flagMode, "replicated", "Service mode (replicated or global)")
	flags.StringVar(&opts.name, flagName, "", "Service name")

	addServiceFlags(flags, opts, buildServiceDefaultFlagMapping())

	flags.VarP(&opts.labels, flagLabel, "l", "Service labels")
	flags.Var(&opts.containerLabels, flagContainerLabel, "Container labels")
	flags.VarP(&opts.env, flagEnv, "e", "Set environment variables")
	flags.Var(&opts.envFile, flagEnvFile, "Read in a file of environment variables")
	flags.Var(&opts.mounts, flagMount, "Attach a filesystem mount to the service")
	flags.Var(&opts.constraints, flagConstraint, "Placement constraints")
	flags.Var(&opts.placementPrefs, flagPlacementPref, "Add a placement preference")
	flags.SetAnnotation(flagPlacementPref, "version", []string{"1.28"})
	flags.Var(&opts.networks, flagNetwork, "Network attachments")
	flags.Var(&opts.secrets, flagSecret, "Specify secrets to expose to the service")
	flags.SetAnnotation(flagSecret, "version", []string{"1.25"})
	flags.Var(&opts.configs, flagConfig, "Specify configurations to expose to the service")
	flags.SetAnnotation(flagConfig, "version", []string{"1.30"})
	flags.VarP(&opts.endpoint.publishPorts, flagPublish, "p", "Publish a port as a node port")
	flags.Var(&opts.groups, flagGroup, "Set one or more supplementary user groups for the container")
	flags.SetAnnotation(flagGroup, "version", []string{"1.25"})
	flags.Var(&opts.dns, flagDNS, "Set custom DNS servers")
	flags.SetAnnotation(flagDNS, "version", []string{"1.25"})
	flags.Var(&opts.dnsOption, flagDNSOption, "Set DNS options")
	flags.SetAnnotation(flagDNSOption, "version", []string{"1.25"})
	flags.Var(&opts.dnsSearch, flagDNSSearch, "Set custom DNS search domains")
	flags.SetAnnotation(flagDNSSearch, "version", []string{"1.25"})
	flags.Var(&opts.hosts, flagHost, "Set one or more custom host-to-IP mappings (host:ip)")
	flags.SetAnnotation(flagHost, "version", []string{"1.25"})
	flags.BoolVar(&opts.init, flagInit, false, "Use an init inside each service container to forward signals and reap processes")
	flags.SetAnnotation(flagInit, "version", []string{"1.37"})
	flags.Var(&opts.sysctls, flagSysCtl, "Sysctl options")
	flags.SetAnnotation(flagSysCtl, "version", []string{"1.40"})

	flags.Var(cliopts.NewListOptsRef(&opts.resources.resGenericResources, ValidateSingleGenericResource), "generic-resource", "User defined resources")
	flags.SetAnnotation(flagHostAdd, "version", []string{"1.32"})

	flags.SetInterspersed(false)
	return cmd
}

func runCreate(dockerCli command.Cli, flags *pflag.FlagSet, opts *serviceOptions) error {
	apiClient := dockerCli.Client()
	createOpts := types.ServiceCreateOptions{}

	ctx := context.Background()

	service, err := opts.ToService(ctx, apiClient, flags)
	if err != nil {
		return err
	}

	if err = validateAPIVersion(service, dockerCli.Client().ClientVersion()); err != nil {
		return err
	}

	specifiedSecrets := opts.secrets.Value()
	if len(specifiedSecrets) > 0 {
		// parse and validate secrets
		secrets, err := ParseSecrets(apiClient, specifiedSecrets)
		if err != nil {
			return err
		}
		service.TaskTemplate.ContainerSpec.Secrets = secrets
	}

	if err := setConfigs(apiClient, &service, opts); err != nil {
		return err
	}

	if err := resolveServiceImageDigestContentTrust(dockerCli, &service); err != nil {
		return err
	}

	// only send auth if flag was set
	if opts.registryAuth {
		// Retrieve encoded auth token from the image reference
		encodedAuth, err := command.RetrieveAuthTokenFromImage(ctx, dockerCli, opts.image)
		if err != nil {
			return err
		}
		createOpts.EncodedRegistryAuth = encodedAuth
	}

	// query registry if flag disabling it was not set
	if !opts.noResolveImage && versions.GreaterThanOrEqualTo(apiClient.ClientVersion(), "1.30") {
		createOpts.QueryRegistry = true
	}

	response, err := apiClient.ServiceCreate(ctx, service, createOpts)
	if err != nil {
		return err
	}

	for _, warning := range response.Warnings {
		fmt.Fprintln(dockerCli.Err(), warning)
	}

	fmt.Fprintf(dockerCli.Out(), "%s\n", response.ID)

	if opts.detach || versions.LessThan(apiClient.ClientVersion(), "1.29") {
		return nil
	}

	return waitOnService(ctx, dockerCli, response.ID, opts.quiet)
}

// setConfigs does double duty: it both sets the ConfigReferences of the
// service, and it sets the service CredentialSpec. This is because there is an
// interplay between the CredentialSpec and the Config it depends on.
func setConfigs(apiClient client.ConfigAPIClient, service *swarm.ServiceSpec, opts *serviceOptions) error {
	specifiedConfigs := opts.configs.Value()
	// if the user has requested to use a Config, for the CredentialSpec add it
	// to the specifiedConfigs as a RuntimeTarget.
	if cs := opts.credentialSpec.Value(); cs != nil && cs.Config != "" {
		specifiedConfigs = append(specifiedConfigs, &swarm.ConfigReference{
			ConfigName: cs.Config,
			Runtime:    &swarm.ConfigReferenceRuntimeTarget{},
		})
	}
	if len(specifiedConfigs) > 0 {
		// parse and validate configs
		configs, err := ParseConfigs(apiClient, specifiedConfigs)
		if err != nil {
			return err
		}
		service.TaskTemplate.ContainerSpec.Configs = configs
		// if we have a CredentialSpec Config, find its ID and rewrite the
		// field on the spec
		//
		// we check the opts instead of the service directly because there are
		// a few layers of nullable objects in the service, which is a PITA
		// to traverse, but the existence of the option implies that those are
		// non-null.
		if cs := opts.credentialSpec.Value(); cs != nil && cs.Config != "" {
			for _, config := range configs {
				if config.ConfigName == cs.Config {
					service.TaskTemplate.ContainerSpec.Privileges.CredentialSpec.Config = config.ConfigID
					// we've found the right config, no need to keep iterating
					// through the rest of them.
					break
				}
			}
		}
	}

	return nil
}
