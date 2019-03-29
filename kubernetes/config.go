package kubernetes

import api "github.com/docker/compose-on-kubernetes/api"

// NewKubernetesConfig resolves the path to the desired Kubernetes configuration file based on
// the KUBECONFIG environment variable and command line flags.
// Deprecated: Use github.com/docker/compose-on-kubernetes/api.NewKubernetesConfig instead
var NewKubernetesConfig = api.NewKubernetesConfig
