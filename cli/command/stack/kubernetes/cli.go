package kubernetes

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/kubernetes"
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	restclient "k8s.io/client-go/rest"
)

// KubeCli holds kubernetes specifics (client, namespace) with the command.Cli
type KubeCli struct {
	command.Cli
	KubeConfig    *restclient.Config
	KubeNamespace string
}

type Options struct {
	Namespace string
	Config    string
}

func NewOptions(flags *flag.FlagSet) Options {
	var opts Options
	if flags.Changed("namespace") {
		if namespace, err := flags.GetString("namespace"); err == nil {
			opts.Namespace = namespace
		}
	}
	if flags.Changed("kubeconfig") {
		if kubeConfig, err := flags.GetString("kubeconfig"); err == nil {
			opts.Config = kubeConfig
		}
	}
	return opts
}

// WrapCli wraps command.Cli with kubernetes specifics
func WrapCli(dockerCli command.Cli, opts Options) (*KubeCli, error) {
	var err error
	cli := &KubeCli{
		Cli:           dockerCli,
		KubeNamespace: "default",
	}
	if opts.Namespace != "" {
		cli.KubeNamespace = opts.Namespace
	}
	kubeConfig := opts.Config
	if kubeConfig == "" {
		if config := os.Getenv("KUBECONFIG"); config != "" {
			kubeConfig = config
		} else {
			kubeConfig = filepath.Join(homedir.Get(), ".kube/config")
		}
	}

	config, err := kubernetes.NewKubernetesConfig(opts.Config)
	if err != nil {
		return nil, err
	}
	cli.KubeConfig = config

	return cli, nil
}

func (c *KubeCli) composeClient() (*Factory, error) {
	return NewFactory(c.KubeNamespace, c.KubeConfig)
}

func (c *KubeCli) stacks() (stackClient, error) {
	version, err := kubernetes.GetAPIVersion(c.KubeConfig)
	if err != nil {
		return nil, err
	}

	switch version {
	case kubernetes.StackAPIV1Beta1:
		return c.newStackV1Beta1()
	case kubernetes.StackAPIV1Beta2:
		return c.newStackV1Beta2()
	default:
		return nil, errors.Errorf("no supported Stack API version")
	}
}
