package kubernetes

import (
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context"
	"github.com/docker/cli/cli/context/store"
	api "github.com/docker/compose-on-kubernetes/api"
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// EndpointMeta is a typed wrapper around a context-store generic endpoint describing
// a Kubernetes endpoint, without TLS data
type EndpointMeta struct {
	context.EndpointMetaBase
	DefaultNamespace string                           `json:",omitempty"`
	AuthProvider     *clientcmdapi.AuthProviderConfig `json:",omitempty"`
	Exec             *clientcmdapi.ExecConfig         `json:",omitempty"`
}

var _ command.EndpointDefaultResolver = &EndpointMeta{}

// Endpoint is a typed wrapper around a context-store generic endpoint describing
// a Kubernetes endpoint, with TLS data
type Endpoint struct {
	EndpointMeta
	TLSData *context.TLSData
}

func init() {
	command.RegisterDefaultStoreEndpoints(
		store.EndpointTypeGetter(KubernetesEndpoint, func() interface{} { return &EndpointMeta{} }),
	)
}

// WithTLSData loads TLS materials for the endpoint
func (c *EndpointMeta) WithTLSData(s store.Reader, contextName string) (Endpoint, error) {
	tlsData, err := context.LoadTLSData(s, contextName, KubernetesEndpoint)
	if err != nil {
		return Endpoint{}, err
	}
	return Endpoint{
		EndpointMeta: *c,
		TLSData:      tlsData,
	}, nil
}

// KubernetesConfig creates the kubernetes client config from the endpoint
func (c *Endpoint) KubernetesConfig() clientcmd.ClientConfig {
	cfg := clientcmdapi.NewConfig()
	cluster := clientcmdapi.NewCluster()
	cluster.Server = c.Host
	cluster.InsecureSkipTLSVerify = c.SkipTLSVerify
	authInfo := clientcmdapi.NewAuthInfo()
	if c.TLSData != nil {
		cluster.CertificateAuthorityData = c.TLSData.CA
		authInfo.ClientCertificateData = c.TLSData.Cert
		authInfo.ClientKeyData = c.TLSData.Key
	}
	authInfo.AuthProvider = c.AuthProvider
	authInfo.Exec = c.Exec
	cfg.Clusters["cluster"] = cluster
	cfg.AuthInfos["authInfo"] = authInfo
	ctx := clientcmdapi.NewContext()
	ctx.AuthInfo = "authInfo"
	ctx.Cluster = "cluster"
	ctx.Namespace = c.DefaultNamespace
	cfg.Contexts["context"] = ctx
	cfg.CurrentContext = "context"
	return clientcmd.NewDefaultClientConfig(*cfg, &clientcmd.ConfigOverrides{})
}

// ResolveDefault returns endpoint metadata for the default Kubernetes
// endpoint, which is derived from the env-based kubeconfig.
func (c *EndpointMeta) ResolveDefault(stackOrchestrator command.Orchestrator) (interface{}, *store.EndpointTLSData, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.Get(), ".kube/config")
	}
	kubeEP, err := FromKubeConfig(kubeconfig, "", "")
	if err != nil {
		if stackOrchestrator == command.OrchestratorKubernetes || stackOrchestrator == command.OrchestratorAll {
			return nil, nil, errors.Wrapf(err, "default orchestrator is %s but unable to resolve kubernetes endpoint", stackOrchestrator)
		}

		// We deliberately quash the error here, returning nil
		// for the first argument is sufficient to indicate we weren't able to
		// provide a default
		return nil, nil, nil
	}

	var tls *store.EndpointTLSData
	if kubeEP.TLSData != nil {
		tls = kubeEP.TLSData.ToStoreTLSData()
	}
	return kubeEP.EndpointMeta, tls, nil
}

// EndpointFromContext extracts kubernetes endpoint info from current context
func EndpointFromContext(metadata store.Metadata) *EndpointMeta {
	ep, ok := metadata.Endpoints[KubernetesEndpoint]
	if !ok {
		return nil
	}
	typed, ok := ep.(EndpointMeta)
	if !ok {
		return nil
	}
	return &typed
}

// ConfigFromContext resolves a kubernetes client config for the specified context.
// If kubeconfigOverride is specified, use this config file instead of the context defaults.ConfigFromContext
// if command.ContextDockerHost is specified as the context name, fallsback to the default user's kubeconfig file
func ConfigFromContext(name string, s store.Reader) (clientcmd.ClientConfig, error) {
	ctxMeta, err := s.GetMetadata(name)
	if err != nil {
		return nil, err
	}
	epMeta := EndpointFromContext(ctxMeta)
	if epMeta != nil {
		ep, err := epMeta.WithTLSData(s, name)
		if err != nil {
			return nil, err
		}
		return ep.KubernetesConfig(), nil
	}
	// context has no kubernetes endpoint
	return api.NewKubernetesConfig(""), nil
}
