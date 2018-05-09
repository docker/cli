package kubernetes

import (
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/kubernetes"
	flag "github.com/spf13/pflag"
	kubeclient "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// KubeCli holds kubernetes specifics (client, namespace) with the command.Cli
type KubeCli struct {
	command.Cli
	kubeConfig    *restclient.Config
	kubeNamespace string
	clientSet     *kubeclient.Clientset
}

// Options contains resolved parameters to initialize kubernetes clients
type Options struct {
	Namespace string
	Config    string
}

// NewOptions returns an Options initialized with command line flags
func NewOptions(flags *flag.FlagSet) Options {
	var opts Options
	if namespace, err := flags.GetString("namespace"); err == nil {
		opts.Namespace = namespace
	}
	if kubeConfig, err := flags.GetString("kubeconfig"); err == nil {
		opts.Config = kubeConfig
	}
	return opts
}

// WrapCli wraps command.Cli with kubernetes specifics
func WrapCli(dockerCli command.Cli, opts Options) (*KubeCli, error) {
	var err error
	cli := &KubeCli{
		Cli: dockerCli,
	}
	clientConfig := kubernetes.NewKubernetesConfig(opts.Config)

	cli.kubeNamespace = opts.Namespace
	if opts.Namespace == "" {
		configNamespace, _, err := clientConfig.Namespace()
		if err != nil {
			return nil, err
		}
		cli.kubeNamespace = configNamespace
	}

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	cli.kubeConfig = config

	clientSet, err := kubeclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	cli.clientSet = clientSet

	return cli, nil
}

func (c *KubeCli) composeClient() (*Factory, error) {
	return NewFactory(c.kubeNamespace, c.kubeConfig, c.clientSet)
}
