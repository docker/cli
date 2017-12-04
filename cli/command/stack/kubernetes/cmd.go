package kubernetes

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	composev1beta1 "github.com/docker/cli/kubernetes/client/clientset_generated/clientset/typed/compose/v1beta1"
	"github.com/docker/docker/pkg/homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// AddStackCommands adds `stack` subcommands
func AddStackCommands(root *cobra.Command, dockerCli command.Cli) {
	var kubeCli kubeCli
	configureCommand(root, &kubeCli)
	root.AddCommand(
		newDeployCommand(dockerCli, &kubeCli),
		newListCommand(dockerCli, &kubeCli),
		newRemoveCommand(dockerCli, &kubeCli),
		newServicesCommand(dockerCli, &kubeCli),
		newPsCommand(dockerCli, &kubeCli),
	)
}

// NewTopLevelDeployCommand returns a command for `docker deploy`
func NewTopLevelDeployCommand(dockerCli command.Cli) *cobra.Command {
	var kubeCli kubeCli
	cmd := newDeployCommand(dockerCli, &kubeCli)
	configureCommand(cmd, &kubeCli)
	return cmd
}

func configureCommand(root *cobra.Command, kubeCli *kubeCli) {
	var (
		kubeOpts kubeOptions
	)
	kubeOpts.installFlags(root.PersistentFlags())
	preRunE := root.PersistentPreRunE
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if preRunE != nil {
			if err := preRunE(cmd, args); err != nil {
				return err
			}
		}
		kubeCli.kubeNamespace = kubeOpts.namespace
		if kubeCli.kubeNamespace == "" {
			kubeCli.kubeNamespace = "default"
		}
		// Read kube config flag and environment variable
		if kubeOpts.kubeconfig == "" {
			if config := os.Getenv("KUBECONFIG"); config != "" {
				kubeOpts.kubeconfig = config
			} else {
				kubeOpts.kubeconfig = filepath.Join(homedir.Get(), ".kube/config")
			}
		}
		config, err := clientcmd.BuildConfigFromFlags("", kubeOpts.kubeconfig)
		if err != nil {
			return err
		}
		kubeCli.kubeConfig = config
		return nil
	}
}

// KubeOptions are options specific to kubernetes
type kubeOptions struct {
	namespace  string
	kubeconfig string
}

// InstallFlags adds flags for the common options on the FlagSet
func (opts *kubeOptions) installFlags(flags *pflag.FlagSet) {
	flags.StringVar(&opts.namespace, "namespace", "default", "Kubernetes namespace to use")
	flags.StringVar(&opts.kubeconfig, "kubeconfig", "", "Kubernetes config file")
}

type kubeCli struct {
	kubeConfig    *restclient.Config
	kubeNamespace string
}

func (c *kubeCli) ComposeClient() (*Factory, error) {
	return NewFactory(c.kubeNamespace, c.kubeConfig)
}

func (c *kubeCli) KubeConfig() *restclient.Config {
	return c.kubeConfig
}

func (c *kubeCli) Stacks() (composev1beta1.StackInterface, error) {
	err := APIPresent(c.kubeConfig)
	if err != nil {
		return nil, err
	}

	clientSet, err := composev1beta1.NewForConfig(c.kubeConfig)
	if err != nil {
		return nil, err
	}

	return clientSet.Stacks(c.kubeNamespace), nil
}
