package kubernetes

import (
	"os"
	"path/filepath"

	"github.com/docker/docker/pkg/homedir"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewKubernetesConfig resolves the path to the desired Kubernetes configuration file, depending
// environment variable and command line flag.
func NewKubernetesConfig(configFlag string) (*restclient.Config, error) {
	kubeConfig := configFlag
	if kubeConfig == "" {
		if config := os.Getenv("KUBECONFIG"); config != "" {
			kubeConfig = config
		} else {
			kubeConfig = filepath.Join(homedir.Get(), ".kube/config")
		}
	}
	return clientcmd.BuildConfigFromFlags("", kubeConfig)
}
