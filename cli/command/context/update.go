package context

import (
	"bytes"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter/tabwriter"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

// updateOptions are the options used to update a context.
type updateOptions struct {
	description string
	endpoint    map[string]string
}

func longUpdateDescription() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("Update a context\n\nDocker endpoint config:\n\n")
	tw := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
	_, _ = fmt.Fprintln(tw, "NAME\tDESCRIPTION")
	for _, d := range dockerConfigKeysDescriptions {
		_, _ = fmt.Fprintf(tw, "%s\t%s\n", d.name, d.description)
	}
	_ = tw.Flush()
	buf.WriteString("\nExample:\n\n$ docker context update my-context --description \"some description\" --docker \"host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file\"\n")
	return buf.String()
}

func newUpdateCommand(dockerCLI command.Cli) *cobra.Command {
	opts := updateOptions{}
	cmd := &cobra.Command{
		Use:   "update [OPTIONS] CONTEXT",
		Short: "Update a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(dockerCLI, args[0], opts)
		},
		Long:                  longUpdateDescription(),
		ValidArgsFunction:     completeContextNames(dockerCLI, 1, false),
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.description, "description", "", "Description of the context")
	flags.StringToStringVar(&opts.endpoint, "docker", nil, "set the docker endpoint")
	return cmd
}

// runUpdate updates a Docker context.
func runUpdate(dockerCLI command.Cli, name string, opts updateOptions) error {
	if err := store.ValidateContextName(name); err != nil {
		return err
	}
	s := dockerCLI.ContextStore()
	c, err := s.GetMetadata(name)
	if err != nil {
		return err
	}
	dockerContext, err := command.GetDockerContext(c)
	if err != nil {
		return err
	}
	if opts.description != "" {
		dockerContext.Description = opts.description
	}

	c.Metadata = dockerContext

	tlsDataToReset := make(map[string]*store.EndpointTLSData)

	if opts.endpoint != nil {
		dockerEP, dockerTLS, err := getDockerEndpointMetadataAndTLS(s, opts.endpoint)
		if err != nil {
			return fmt.Errorf("unable to create docker endpoint config: %w", err)
		}
		c.Endpoints[docker.DockerEndpoint] = dockerEP
		tlsDataToReset[docker.DockerEndpoint] = dockerTLS
	}
	if err := validateEndpoints(c); err != nil {
		return err
	}
	if err := s.CreateOrUpdate(c); err != nil {
		return err
	}
	for ep, tlsData := range tlsDataToReset {
		if err := s.ResetEndpointTLSMaterial(name, ep, tlsData); err != nil {
			return err
		}
	}

	_, _ = fmt.Fprintln(dockerCLI.Out(), name)
	_, _ = fmt.Fprintf(dockerCLI.Err(), "Successfully updated context %q\n", name)
	return nil
}

func validateEndpoints(c store.Metadata) error {
	_, err := command.GetDockerContext(c)
	return err
}
