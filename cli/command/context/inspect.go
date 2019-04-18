package context

import (
	"errors"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/inspect"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	format string
	refs   []string
}

// newInspectCommand creates a new cobra.Command for `docker image inspect`
func newInspectCommand(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions

	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] [CONTEXT] [CONTEXT...]",
		Short: "Display detailed information on one or more contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.refs = args
			if len(opts.refs) == 0 {
				if dockerCli.CurrentContext() == "" {
					return errors.New("no context specified")
				}
				opts.refs = []string{dockerCli.CurrentContext()}
			}
			return runInspect(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "", "Format the output using the given Go template")
	return cmd
}

func runInspect(dockerCli command.Cli, opts inspectOptions) error {
	getRefFunc := func(ref string) (interface{}, []byte, error) {
		c, err := dockerCli.ContextStore().GetMetadata(ref)
		if err != nil {
			return nil, nil, err
		}
		tlsListing, err := dockerCli.ContextStore().ListTLSFiles(ref)
		if err != nil {
			return nil, nil, err
		}
		return contextWithTLSListing{
			Metadata:    c,
			TLSMaterial: tlsListing,
			Storage:     dockerCli.ContextStore().GetStorageInfo(ref),
		}, nil, nil
	}
	return inspect.Inspect(dockerCli.Out(), opts.refs, opts.format, getRefFunc)
}

type contextWithTLSListing struct {
	store.Metadata
	TLSMaterial map[string]store.EndpointFiles
	Storage     store.StorageInfo
}
