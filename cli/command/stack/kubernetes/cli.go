package kubernetes

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/kubernetes"
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
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
		Cli:           dockerCli,
		kubeNamespace: "default",
	}
	if opts.Namespace != "" {
		cli.kubeNamespace = opts.Namespace
	}
	kubeConfig := opts.Config
	if kubeConfig == "" {
		if config := os.Getenv("KUBECONFIG"); config != "" {
			kubeConfig = config
		} else {
			kubeConfig = filepath.Join(homedir.Get(), ".kube/config")
		}
	}
	config, err := kubernetes.NewKubernetesConfig(kubeConfig)
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
	return NewFactory(c.kubeNamespace, c.kubeConfig)
}

func (c *KubeCli) stacks() (stackClient, error) {
	version, err := kubernetes.GetStackAPIVersion(c.clientSet)
	if err != nil {
		return nil, err
	}

	switch version {
	case kubernetes.StackAPIV1Beta1:
		return newStackV1Beta1(c.kubeConfig, c.kubeNamespace)
	case kubernetes.StackAPIV1Beta2:
		return newStackV1Beta2(c.kubeConfig, c.kubeNamespace)
	default:
		return nil, errors.Errorf("no supported Stack API version")
	}
}
