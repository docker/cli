package context

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/docker/pkg/homedir"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

type useOptions struct {
	skipKubeconfig bool
}

func newUseCommand(dockerCli command.Cli) *cobra.Command {
	opts := &useOptions{}
	cmd := &cobra.Command{
		Use:   "use CONTEXT [OPTIONS]",
		Short: "Set the current docker context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return runUse(dockerCli, name, *opts)
		},
	}
	cmd.Flags().BoolVar(&opts.skipKubeconfig, "skip-kubeconfig", false, "Do not modify current kubeconfig file (set via KUBECONFIG environment variable, or ~/.kube/config)")
	return cmd
}

func runUse(dockerCli command.Cli, name string, opts useOptions) error {
	if err := validateContextName(name); err != nil && name != "default" {
		return err
	}
	ctxMeta, err := dockerCli.ContextStore().GetContextMetadata(name)
	if err != nil && name != "default" {
		return err
	}
	configValue := name
	if configValue == "default" {
		configValue = ""
	}
	if !opts.skipKubeconfig {
		if err := applyToKubeconfig(dockerCli, configValue, ctxMeta); err != nil {
			return err
		}
	}
	dockerConfig := dockerCli.ConfigFile()
	dockerConfig.CurrentContext = configValue
	if err := dockerConfig.Save(); err != nil {
		return err
	}
	fmt.Fprintln(dockerCli.Out(), name)
	fmt.Fprintf(dockerCli.Err(), "Current context is now %q\n", name)
	return nil
}

func applyToKubeconfig(dockerCli command.Cli, contextName string, contextMeta store.ContextMetadata) error {
	if contextName == "" {
		return nil
	}
	kubeEndpointMeta := kubernetes.EndpointFromContext(contextMeta)
	if kubeEndpointMeta == nil {
		return nil
	}
	kubeEndpoint, err := kubeEndpointMeta.WithTLSData(dockerCli.ContextStore(), contextName)
	if err != nil {
		return err
	}
	kubeconfig, err := kubeEndpoint.KubernetesNamedConfig(contextName, contextName, contextName).RawConfig()
	if err != nil {
		return err
	}
	kubeconfigPath := kubeconfigPath()
	_, err = os.Stat(kubeconfigPath)
	switch {
	case os.IsNotExist(err):
		return clientcmd.WriteToFile(kubeconfig, kubeconfigPath)
	case err != nil:
		return err
	}
	targetConfig, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return err
	}
	targetConfig.AuthInfos[contextName] = kubeconfig.AuthInfos[contextName]
	targetConfig.Clusters[contextName] = kubeconfig.Clusters[contextName]
	targetConfig.Contexts[contextName] = kubeconfig.Contexts[contextName]
	targetConfig.CurrentContext = contextName
	return clientcmd.WriteToFile(*targetConfig, kubeconfigPath)
}

func kubeconfigPath() string {
	if config := os.Getenv("KUBECONFIG"); config != "" {
		return config
	}
	return filepath.Join(homedir.Get(), ".kube/config")
}
