package context

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/context/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	name                     string
	description              string
	defaultStackOrchestrator string
	docker                   map[string]string
	kubernetes               map[string]string
}

func longUpdateDescription() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("Update a context\n\nDocker endpoint config:\n\n")
	tw := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
	fmt.Fprintln(tw, "NAME\tDESCRIPTION")
	for _, d := range dockerConfigKeysDescriptions {
		fmt.Fprintf(tw, "%s\t%s\n", d.name, d.description)
	}
	tw.Flush()
	buf.WriteString("\nKubernetes endpoint config:\n\n")
	tw = tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
	fmt.Fprintln(tw, "NAME\tDESCRIPTION")
	for _, d := range kubernetesConfigKeysDescriptions {
		fmt.Fprintf(tw, "%s\t%s\n", d.name, d.description)
	}
	tw.Flush()
	buf.WriteString("\nExample:\n\n$ docker context update my-context --description \"some description\" --docker \"host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file\"\n")
	return buf.String()
}

func newUpdateCommand(dockerCli command.Cli) *cobra.Command {
	opts := &updateOptions{}
	cmd := &cobra.Command{
		Use:   "update [OPTIONS] CONTEXT",
		Short: "Update a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runUpdate(dockerCli, opts)
		},
		Long: longUpdateDescription(),
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.description, "description", "", "Description of the context")
	flags.StringVar(
		&opts.defaultStackOrchestrator,
		"default-stack-orchestrator", "",
		"Default orchestrator for stack operations to use with this context (swarm|kubernetes|all)")
	flags.StringToStringVar(&opts.docker, "docker", nil, "set the docker endpoint")
	flags.StringToStringVar(&opts.kubernetes, "kubernetes", nil, "set the kubernetes endpoint")
	return cmd
}

func runUpdate(cli command.Cli, o *updateOptions) error {
	if err := validateContextName(o.name); err != nil {
		return err
	}
	s := cli.ContextStore()
	c, err := s.GetContextMetadata(o.name)
	if err != nil {
		return err
	}
	dockerContext, err := command.GetDockerContext(c)
	if err != nil {
		return err
	}
	if o.defaultStackOrchestrator != "" {
		stackOrchestrator, err := command.NormalizeOrchestrator(o.defaultStackOrchestrator)
		if err != nil {
			return errors.Wrap(err, "unable to parse default-stack-orchestrator")
		}
		dockerContext.StackOrchestrator = stackOrchestrator
	}
	if o.description != "" {
		dockerContext.Description = o.description
	}

	c.Metadata = dockerContext

	tlsDataToReset := make(map[string]*store.EndpointTLSData)

	if o.docker != nil {
		dockerEP, dockerTLS, err := getDockerEndpointMetadataAndTLS(cli, o.docker)
		if err != nil {
			return errors.Wrap(err, "unable to create docker endpoint config")
		}
		c.Endpoints[docker.DockerEndpoint] = dockerEP
		tlsDataToReset[docker.DockerEndpoint] = dockerTLS
	}
	if o.kubernetes != nil {
		kubernetesEP, kubernetesTLS, err := getKubernetesEndpointMetadataAndTLS(cli, o.kubernetes)
		if err != nil {
			return errors.Wrap(err, "unable to create kubernetes endpoint config")
		}
		if kubernetesEP == nil {
			delete(c.Endpoints, kubernetes.KubernetesEndpoint)
		} else {
			c.Endpoints[kubernetes.KubernetesEndpoint] = kubernetesEP
			tlsDataToReset[kubernetes.KubernetesEndpoint] = kubernetesTLS
		}
	}
	if err := validateEndpointsAndOrchestrator(c); err != nil {
		return err
	}
	if err := s.CreateOrUpdateContext(c); err != nil {
		return err
	}
	for ep, tlsData := range tlsDataToReset {
		if err := s.ResetContextEndpointTLSMaterial(o.name, ep, tlsData); err != nil {
			return err
		}
	}

	fmt.Fprintln(cli.Out(), o.name)
	fmt.Fprintf(cli.Err(), "Successfully updated context %q\n", o.name)
	return nil
}

func validateEndpointsAndOrchestrator(c store.ContextMetadata) error {
	dockerContext, err := command.GetDockerContext(c)
	if err != nil {
		return err
	}
	if _, ok := c.Endpoints[kubernetes.KubernetesEndpoint]; !ok && dockerContext.StackOrchestrator.HasKubernetes() {
		return errors.Errorf("cannot specify orchestrator %q without configuring a Kubernetes endpoint", dockerContext.StackOrchestrator)
	}
	return nil
}
