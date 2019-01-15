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
		if ref == "default" {
			return nil, nil, errors.New(`context "default" cannot be inspected`)
		}
		c, err := dockerCli.ContextStore().GetContextMetadata(ref)
		if err != nil {
			return nil, nil, err
		}
		tlsListing, err := dockerCli.ContextStore().ListContextTLSFiles(ref)
		if err != nil {
			return nil, nil, err
		}
		return contextWithTLSListing{
			ContextMetadata: c,
			TLSMaterial:     tlsListing,
			Storage:         dockerCli.ContextStore().GetContextStorageInfo(ref),
		}, nil, nil
	}
	return inspect.Inspect(dockerCli.Out(), opts.refs, opts.format, getRefFunc)
}

type contextWithTLSListing struct {
	store.ContextMetadata
	TLSMaterial map[string]store.EndpointFiles
	Storage     store.ContextStorageInfo
}
