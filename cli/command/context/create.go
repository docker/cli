// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package context

import (
	"bytes"
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/completion"
	"github.com/docker/cli/cli/command/formatter/tabwriter"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/docker/errdefs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// CreateOptions are the options used for creating a context
type CreateOptions struct {
	Name        string
	Description string
	Docker      map[string]string
	From        string
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
		Long:              longCreateDescription(),
		ValidArgsFunction: completion.NoComplete,
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.Description, "description", "", "Description of the context")
	flags.String(
		"default-stack-orchestrator", "",
		`Default orchestrator for stack operations to use with this context ("swarm", "kubernetes", "all")`,
	)
	flags.SetAnnotation("default-stack-orchestrator", "deprecated", nil)
	flags.SetAnnotation("default-stack-orchestrator", "deprecated", nil)
	flags.MarkDeprecated("default-stack-orchestrator", "option will be ignored")
	flags.StringToStringVar(&opts.Docker, "docker", nil, "set the docker endpoint")
	flags.StringToString("kubernetes", nil, "set the kubernetes endpoint")
	flags.SetAnnotation("kubernetes", "kubernetes", nil)
	flags.SetAnnotation("kubernetes", "deprecated", nil)
	flags.MarkDeprecated("kubernetes", "option will be ignored")
	flags.StringVar(&opts.From, "from", "", "create context from a named context")
	return cmd
}

// RunCreate creates a Docker context
func RunCreate(cli command.Cli, o *CreateOptions) error {
	s := cli.ContextStore()
	err := checkContextNameForCreation(s, o.Name)
	if err != nil {
		return err
	}
	switch {
	case o.From == "" && o.Docker == nil:
		err = createFromExistingContext(s, cli.CurrentContext(), o)
	case o.From != "":
		err = createFromExistingContext(s, o.From, o)
	default:
		err = createNewContext(o, cli, s)
	}
	if err == nil {
		fmt.Fprintln(cli.Out(), o.Name)
		fmt.Fprintf(cli.Err(), "Successfully created context %q\n", o.Name)
	}
	return err
}

func createNewContext(o *CreateOptions, cli command.Cli, s store.Writer) error {
	if o.Docker == nil {
		return errors.New("docker endpoint configuration is required")
	}
	dockerEP, dockerTLS, err := getDockerEndpointMetadataAndTLS(cli, o.Docker)
	if err != nil {
		return errors.Wrap(err, "unable to create docker endpoint config")
	}
	contextMetadata := store.Metadata{
		Endpoints: map[string]interface{}{
			docker.DockerEndpoint: dockerEP,
		},
		Metadata: command.DockerContext{
			Description: o.Description,
		},
		Name: o.Name,
	}
	contextTLSData := store.ContextTLSData{}
	if dockerTLS != nil {
		contextTLSData.Endpoints = map[string]store.EndpointTLSData{
			docker.DockerEndpoint: *dockerTLS,
		}
	}
	if err := validateEndpoints(contextMetadata); err != nil {
		return err
	}
	if err := s.CreateOrUpdate(contextMetadata); err != nil {
		return err
	}
	return s.ResetTLSMaterial(o.Name, &contextTLSData)
}

func checkContextNameForCreation(s store.Reader, name string) error {
	if err := store.ValidateContextName(name); err != nil {
		return err
	}
	if _, err := s.GetMetadata(name); !errdefs.IsNotFound(err) {
		if err != nil {
			return errors.Wrap(err, "error while getting existing contexts")
		}
		return errors.Errorf("context %q already exists", name)
	}
	return nil
}

func createFromExistingContext(s store.ReaderWriter, fromContextName string, o *CreateOptions) error {
	if len(o.Docker) != 0 {
		return errors.New("cannot use --docker flag when --from is set")
	}
	reader := store.Export(fromContextName, &descriptionDecorator{
		Reader:      s,
		description: o.Description,
	})
	defer reader.Close()
	return store.Import(o.Name, s, reader)
}

type descriptionDecorator struct {
	store.Reader
	description string
}

func (d *descriptionDecorator) GetMetadata(name string) (store.Metadata, error) {
	c, err := d.Reader.GetMetadata(name)
	if err != nil {
		return c, err
	}
	typedContext, err := command.GetDockerContext(c)
	if err != nil {
		return c, err
	}
	if d.description != "" {
		typedContext.Description = d.description
	}
	c.Metadata = typedContext
	return c, nil
}
