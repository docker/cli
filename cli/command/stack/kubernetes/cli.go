package kubernetes

import (
	"fmt"
	"net"
	"net/url"

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

// AddNamespaceFlag adds the namespace flag to the given flag set
func AddNamespaceFlag(flags *flag.FlagSet) {
	flags.String("namespace", "", "Kubernetes namespace to use")
	flags.SetAnnotation("namespace", "kubernetes", nil)
}

// WrapCli wraps command.Cli with kubernetes specifics
func WrapCli(dockerCli command.Cli, opts Options) (*KubeCli, error) {
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

	if dockerCli.ClientInfo().HasAll() {
		if err := cli.checkHostsMatch(); err != nil {
			return nil, err
		}
	}
	return cli, nil
}

func (c *KubeCli) composeClient() (*Factory, error) {
	return NewFactory(c.kubeNamespace, c.kubeConfig, c.clientSet)
}

func (c *KubeCli) checkHostsMatch() error {
	daemonEndpoint, err := url.Parse(c.Client().DaemonHost())
	if err != nil {
		return err
	}
	kubeEndpoint, err := url.Parse(c.kubeConfig.Host)
	if err != nil {
		return err
	}
	if daemonEndpoint.Hostname() == kubeEndpoint.Hostname() {
		return nil
	}
	// The daemon can be local in Docker for Desktop, e.g. "npipe", "unix", ...
	if daemonEndpoint.Scheme != "tcp" {
		ips, err := net.LookupIP(kubeEndpoint.Hostname())
		if err != nil {
			return err
		}
		for _, ip := range ips {
			if ip.IsLoopback() {
				return nil
			}
		}
	}
	fmt.Fprintf(c.Err(), "WARNING: Swarm and Kubernetes hosts do not match (docker host=%s, kubernetes host=%s).\n"+
		"         Update $DOCKER_HOST (or pass -H), or use 'kubectl config use-context' to match.\n", daemonEndpoint.Hostname(), kubeEndpoint.Hostname())
	return nil
}
