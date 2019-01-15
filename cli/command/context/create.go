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

type createOptions struct {
	name                     string
	description              string
	defaultStackOrchestrator string
	docker                   map[string]string
	kubernetes               map[string]string
}

func longCreateDescription() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("Create a context\n\nDocker endpoint config:\n\n")
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
	buf.WriteString("\nExample:\n\n$ docker context create my-context --description \"some description\" --docker \"host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file\"\n")
	return buf.String()
}

func newCreateCommand(dockerCli command.Cli) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create [OPTIONS] CONTEXT",
		Short: "Create a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runCreate(dockerCli, opts)
		},
		Long: longCreateDescription(),
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

func runCreate(cli command.Cli, o *createOptions) error {
	s := cli.ContextStore()
	if err := checkContextNameForCreation(s, o.name); err != nil {
		return err
	}
	stackOrchestrator, err := command.NormalizeOrchestrator(o.defaultStackOrchestrator)
	if err != nil {
		return errors.Wrap(err, "unable to parse default-stack-orchestrator")
	}
	contextMetadata := store.ContextMetadata{
		Endpoints: make(map[string]interface{}),
		Metadata: command.DockerContext{
			Description:       o.description,
			StackOrchestrator: stackOrchestrator,
		},
		Name: o.name,
	}
	if o.docker == nil {
		return errors.New("docker endpoint configuration is required")
	}
	contextTLSData := store.ContextTLSData{
		Endpoints: make(map[string]store.EndpointTLSData),
	}
	dockerEP, dockerTLS, err := getDockerEndpointMetadataAndTLS(cli, o.docker)
	if err != nil {
		return errors.Wrap(err, "unable to create docker endpoint config")
	}
	contextMetadata.Endpoints[docker.DockerEndpoint] = dockerEP
	if dockerTLS != nil {
		contextTLSData.Endpoints[docker.DockerEndpoint] = *dockerTLS
	}
	if o.kubernetes != nil {
		kubernetesEP, kubernetesTLS, err := getKubernetesEndpointMetadataAndTLS(cli, o.kubernetes)
		if err != nil {
			return errors.Wrap(err, "unable to create kubernetes endpoint config")
		}
		if kubernetesEP == nil && stackOrchestrator.HasKubernetes() {
			return errors.Errorf("cannot specify orchestrator %q without configuring a Kubernetes endpoint", stackOrchestrator)
		}
		if kubernetesEP != nil {
			contextMetadata.Endpoints[kubernetes.KubernetesEndpoint] = kubernetesEP
		}
		if kubernetesTLS != nil {
			contextTLSData.Endpoints[kubernetes.KubernetesEndpoint] = *kubernetesTLS
		}
	}
	if err := validateEndpointsAndOrchestrator(contextMetadata); err != nil {
		return err
	}
	if err := s.CreateOrUpdateContext(contextMetadata); err != nil {
		return err
	}
	if err := s.ResetContextTLSMaterial(o.name, &contextTLSData); err != nil {
		return err
	}
	fmt.Fprintln(cli.Out(), o.name)
	fmt.Fprintf(cli.Err(), "Successfully created context %q\n", o.name)
	return nil
}

func checkContextNameForCreation(s store.Store, name string) error {
	if err := validateContextName(name); err != nil {
		return err
	}
	if _, err := s.GetContextMetadata(name); !store.IsErrContextDoesNotExist(err) {
		if err != nil {
			return errors.Wrap(err, "error while getting existing contexts")
		}
		return errors.Errorf("context %q already exists", name)
	}
	return nil
}
