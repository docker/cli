package context

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

// ExportOptions are the options used for exporting a context
type ExportOptions struct {
	Kubeconfig  bool
	ContextName string
	Dest        string
}

func newExportCommand(dockerCli command.Cli) *cobra.Command {
	opts := &ExportOptions{}
	cmd := &cobra.Command{
		Use:   "export [OPTIONS] CONTEXT [FILE|-]",
		Short: "Export a context to a tar or kubeconfig file",
		Args:  cli.RequiresRangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ContextName = args[0]
			if len(args) == 2 {
				opts.Dest = args[1]
			} else {
				opts.Dest = opts.ContextName
				if opts.Kubeconfig {
					opts.Dest += ".kubeconfig"
				} else {
					opts.Dest += ".dockercontext"
				}
			}
			return RunExport(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.Kubeconfig, "kubeconfig", false, "Export as a kubeconfig file")
	return cmd
}

func writeTo(dockerCli command.Cli, reader io.Reader, dest string) error {
	var writer io.Writer
	var printDest bool
	if dest == "-" {
		if dockerCli.Out().IsTerminal() {
			return errors.New("cowardly refusing to export to a terminal, please specify a file path")
		}
		writer = dockerCli.Out()
	} else {
		f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
		printDest = true
	}
	if _, err := io.Copy(writer, reader); err != nil {
		return err
	}
	if printDest {
		fmt.Fprintf(dockerCli.Err(), "Written file %q\n", dest)
	}
	return nil
}

// RunExport exports a Docker context
func RunExport(dockerCli command.Cli, opts *ExportOptions) error {
	if err := validateContextName(opts.ContextName); err != nil {
		return err
	}
	ctxMeta, err := dockerCli.ContextStore().GetContextMetadata(opts.ContextName)
	if err != nil {
		return err
	}
	if !opts.Kubeconfig {
		reader := store.Export(opts.ContextName, dockerCli.ContextStore())
		defer reader.Close()
		return writeTo(dockerCli, reader, opts.Dest)
	}
	kubernetesEndpointMeta := kubernetes.EndpointFromContext(ctxMeta)
	if kubernetesEndpointMeta == nil {
		return fmt.Errorf("context %q has no kubernetes endpoint", opts.ContextName)
	}
	kubernetesEndpoint, err := kubernetesEndpointMeta.WithTLSData(dockerCli.ContextStore(), opts.ContextName)
	if err != nil {
		return err
	}
	kubeConfig := kubernetesEndpoint.KubernetesConfig()
	rawCfg, err := kubeConfig.RawConfig()
	if err != nil {
		return err
	}
	data, err := clientcmd.Write(rawCfg)
	if err != nil {
		return err
	}
	return writeTo(dockerCli, bytes.NewBuffer(data), opts.Dest)
}
