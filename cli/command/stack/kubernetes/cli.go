package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	composev1beta1 "github.com/docker/cli/kubernetes/client/clientset/typed/compose/v1beta1"
	"github.com/docker/docker/pkg/homedir"
	"github.com/spf13/cobra"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeCli holds kubernetes specifics (client, namespace) with the command.Cli
type KubeCli struct {
	command.Cli
	KubeConfig    *restclient.Config
	KubeNamespace string
}

// WrapCli wraps command.Cli with kubernetes specifics
func WrapCli(dockerCli command.Cli, cmd *cobra.Command) (*KubeCli, error) {
	var err error
	cli := &KubeCli{
		Cli:           dockerCli,
		KubeNamespace: "default",
	}
	if cmd.Flags().Changed("namespace") {
		cli.KubeNamespace, err = cmd.Flags().GetString("namespace")
		if err != nil {
			return nil, err
		}
	}
	kubeConfig := ""
	if cmd.Flags().Changed("kubeconfig") {
		kubeConfig, err = cmd.Flags().GetString("kubeconfig")
		if err != nil {
			return nil, err
		}
	}
	if kubeConfig == "" {
		if config := os.Getenv("KUBECONFIG"); config != "" {
			kubeConfig = config
		} else {
			kubeConfig = filepath.Join(homedir.Get(), ".kube/config")
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to load kubernetes configuration file '%s'", kubeConfig)
	}
	cli.KubeConfig = config

	return cli, nil
}

func (c *KubeCli) composeClient() (*Factory, error) {
	return NewFactory(c.KubeNamespace, c.KubeConfig)
}

func (c *KubeCli) stacks() (composev1beta1.StackInterface, error) {
	_, err := c.GetAPIVersion()
	if err != nil {
		return nil, err
	}

	clientSet, err := composev1beta1.NewForConfig(c.KubeConfig)
	if err != nil {
		return nil, err
	}

	return clientSet.Stacks(c.KubeNamespace), nil
}
