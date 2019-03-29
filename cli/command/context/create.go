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

// CreateOptions are the options used for creating a context
type CreateOptions struct {
	Name                     string
	Description              string
	DefaultStackOrchestrator string
	Docker                   map[string]string
	Kubernetes               map[string]string
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
	opts := &CreateOptions{}
	cmd := &cobra.Command{
		Use:   "create [OPTIONS] CONTEXT",
		Short: "Create a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return RunCreate(dockerCli, opts)
		},
		Long: longCreateDescription(),
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.Description, "description", "", "Description of the context")
	flags.StringVar(
		&opts.DefaultStackOrchestrator,
		"default-stack-orchestrator", "",
		"Default orchestrator for stack operations to use with this context (swarm|kubernetes|all)")
	flags.StringToStringVar(&opts.Docker, "docker", nil, "set the docker endpoint")
	flags.StringToStringVar(&opts.Kubernetes, "kubernetes", nil, "set the kubernetes endpoint")
	return cmd
}

// RunCreate creates a Docker context
func RunCreate(cli command.Cli, o *CreateOptions) error {
	s := cli.ContextStore()
	if err := checkContextNameForCreation(s, o.Name); err != nil {
		return err
	}
	stackOrchestrator, err := command.NormalizeOrchestrator(o.DefaultStackOrchestrator)
	if err != nil {
		return errors.Wrap(err, "unable to parse default-stack-orchestrator")
	}
	contextMetadata := store.ContextMetadata{
		Endpoints: make(map[string]interface{}),
		Metadata: command.DockerContext{
			Description:       o.Description,
			StackOrchestrator: stackOrchestrator,
		},
		Name: o.Name,
	}
	if o.Docker == nil {
		return errors.New("docker endpoint configuration is required")
	}
	contextTLSData := store.ContextTLSData{
		Endpoints: make(map[string]store.EndpointTLSData),
	}
	dockerEP, dockerTLS, err := getDockerEndpointMetadataAndTLS(cli, o.Docker)
	if err != nil {
		return errors.Wrap(err, "unable to create docker endpoint config")
	}
	contextMetadata.Endpoints[docker.DockerEndpoint] = dockerEP
	if dockerTLS != nil {
		contextTLSData.Endpoints[docker.DockerEndpoint] = *dockerTLS
	}
	if o.Kubernetes != nil {
		kubernetesEP, kubernetesTLS, err := getKubernetesEndpointMetadataAndTLS(cli, o.Kubernetes)
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
	if err := s.ResetContextTLSMaterial(o.Name, &contextTLSData); err != nil {
		return err
	}
	fmt.Fprintln(cli.Out(), o.Name)
	fmt.Fprintf(cli.Err(), "Successfully created context %q\n", o.Name)
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
