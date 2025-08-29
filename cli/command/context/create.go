// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package context

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter/tabwriter"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

// CreateOptions are the options used for creating a context
//
// Deprecated: this type was for internal use and will be removed in the next release.
type CreateOptions struct {
	Name        string
	Description string
	Docker      map[string]string
	From        string

	// Additional Metadata to store in the context. This option is not
	// currently exposed to the user.
	metaData map[string]any
}

// createOptions are the options used for creating a context
type createOptions struct {
	name        string
	description string
	endpoint    map[string]string
	from        string

	// Additional Metadata to store in the context. This option is not
	// currently exposed to the user.
	metaData map[string]any
}

func longCreateDescription() string {
	buf := bytes.NewBuffer(nil)
	buf.WriteString("Create a context\n\nDocker endpoint config:\n\n")
	tw := tabwriter.NewWriter(buf, 20, 1, 3, ' ', 0)
	_, _ = fmt.Fprintln(tw, "NAME\tDESCRIPTION")
	for _, d := range dockerConfigKeysDescriptions {
		_, _ = fmt.Fprintf(tw, "%s\t%s\n", d.name, d.description)
	}
	_ = tw.Flush()
	buf.WriteString("\nExample:\n\n$ docker context create my-context --description \"some description\" --docker \"host=tcp://myserver:2376,ca=~/ca-file,cert=~/cert-file,key=~/key-file\"\n")
	return buf.String()
}

func newCreateCommand(dockerCLI command.Cli) *cobra.Command {
	opts := createOptions{}
	cmd := &cobra.Command{
		Use:   "create [OPTIONS] CONTEXT",
		Short: "Create a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return runCreate(dockerCLI, &opts)
		},
		Long:                  longCreateDescription(),
		ValidArgsFunction:     cobra.NoFileCompletions,
		DisableFlagsInUseLine: true,
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.description, "description", "", "Description of the context")
	flags.StringToStringVar(&opts.endpoint, "docker", nil, "set the docker endpoint")
	flags.StringVar(&opts.from, "from", "", "create context from a named context")
	return cmd
}

// RunCreate creates a Docker context

// Deprecated: this function was for internal use and will be removed in the next release.
func RunCreate(dockerCLI command.Cli, o *CreateOptions) error {
	if o == nil {
		o = &CreateOptions{}
	}

	return runCreate(dockerCLI, &createOptions{
		name:        o.Name,
		description: o.Description,
		endpoint:    o.Docker,
		metaData:    o.metaData,
	})
}

// runCreate creates a Docker context
func runCreate(dockerCLI command.Cli, opts *createOptions) error {
	s := dockerCLI.ContextStore()
	err := checkContextNameForCreation(s, opts.name)
	if err != nil {
		return err
	}
	switch {
	case opts.from == "" && opts.endpoint == nil:
		err = createFromExistingContext(s, dockerCLI.CurrentContext(), opts)
	case opts.from != "":
		err = createFromExistingContext(s, opts.from, opts)
	default:
		err = createNewContext(s, opts)
	}
	if err == nil {
		_, _ = fmt.Fprintln(dockerCLI.Out(), opts.name)
		_, _ = fmt.Fprintf(dockerCLI.Err(), "Successfully created context %q\n", opts.name)
	}
	return err
}

func createNewContext(contextStore store.ReaderWriter, opts *createOptions) error {
	if opts.endpoint == nil {
		return errors.New("docker endpoint configuration is required")
	}
	dockerEP, dockerTLS, err := getDockerEndpointMetadataAndTLS(contextStore, opts.endpoint)
	if err != nil {
		return fmt.Errorf("unable to create docker endpoint config: %w", err)
	}
	contextMetadata := store.Metadata{
		Endpoints: map[string]any{
			docker.DockerEndpoint: dockerEP,
		},
		Metadata: command.DockerContext{
			Description:      opts.description,
			AdditionalFields: opts.metaData,
		},
		Name: opts.name,
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
	if err := contextStore.CreateOrUpdate(contextMetadata); err != nil {
		return err
	}
	return contextStore.ResetTLSMaterial(opts.name, &contextTLSData)
}

func checkContextNameForCreation(s store.Reader, name string) error {
	if err := store.ValidateContextName(name); err != nil {
		return err
	}
	if _, err := s.GetMetadata(name); !errdefs.IsNotFound(err) {
		if err != nil {
			return fmt.Errorf("error while getting existing contexts: %w", err)
		}
		return fmt.Errorf("context %q already exists", name)
	}
	return nil
}

func createFromExistingContext(s store.ReaderWriter, fromContextName string, opts *createOptions) error {
	if len(opts.endpoint) != 0 {
		return errors.New("cannot use --docker flag when --from is set")
	}
	reader := store.Export(fromContextName, &descriptionDecorator{
		Reader:      s,
		description: opts.description,
	})
	defer reader.Close()
	return store.Import(opts.name, s, reader)
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
