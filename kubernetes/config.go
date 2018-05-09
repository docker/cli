package kubernetes

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/homedir"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubernetesConfig resolves the path to the desired Kubernetes configuration file based on
// the KUBECONFIG environment variable and command line flags.
func NewKubernetesConfig(configPath string) clientcmd.ClientConfig {
	kubeConfig := configPath
	if kubeConfig == "" {
		if config := os.Getenv("KUBECONFIG"); config != "" {
			kubeConfig = config
		} else {
			kubeConfig = filepath.Join(homedir.Get(), ".kube/config")
		}
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfig},
		&clientcmd.ConfigOverrides{})
}
