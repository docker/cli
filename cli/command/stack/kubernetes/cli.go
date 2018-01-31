package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
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

func (c *KubeCli) stacks() (stackClient, error) {
	version, err := c.GetAPIVersion()
	if err != nil {
		return nil, err
	}

	switch version {
	case KubernetesStackAPIV1Beta1:
		return c.newStackV1Beta1()
	case KubernetesStackAPIV1Beta2:
		return c.newStackV1Beta2()
	default:
		return nil, fmt.Errorf("could not find matching Stack API version")
	}
}
