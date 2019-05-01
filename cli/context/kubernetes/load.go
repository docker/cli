package kubernetes

import (
	"github.com/docker/cli/cli/context"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/cli/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const tokenKey = "token"

// EndpointMeta is a typed wrapper around a context-store generic endpoint describing
// a Kubernetes endpoint, without TLS data
type EndpointMeta struct {
	context.EndpointMetaBase
	DefaultNamespace string                           `json:",omitempty"`
	AuthProvider     *clientcmdapi.AuthProviderConfig `json:",omitempty"`
	Exec             *clientcmdapi.ExecConfig         `json:",omitempty"`
}

// Endpoint is a typed wrapper around a context-store generic endpoint describing
// a Kubernetes endpoint, with TLS data
type Endpoint struct {
	EndpointMeta
	TLSData *TLSData
}

// TLSData holds sensitive Kubernetes endpoint configurations (ca/cert/key and tokens)
type TLSData struct {
	context.TLSData
	Token string
}

// ToStoreTLSData converts TLSData to the store representation
func (data *TLSData) ToStoreTLSData() *store.EndpointTLSData {
	if data == nil {
		return nil
	}
	result := data.TLSData.ToStoreTLSData()
	if data.Token != "" {
		result.Files[tokenKey] = []byte(data.Token)
	}
	return result
}

// LoadTLSData loads TLS data from the store
func LoadTLSData(s store.Reader, contextName string) (*TLSData, error) {
	tlsData, err := context.LoadTLSData(s, contextName, KubernetesEndpoint)
	if err != nil {
		return nil, err
	}
	if tlsData == nil {
		return nil, nil
	}
	result := &TLSData{
		TLSData: *tlsData,
	}
	token, err := s.GetTLSData(contextName, KubernetesEndpoint, tokenKey)
	if store.IsErrTLSDataDoesNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	result.Token = string(token)
	return result, nil
}

// WithTLSData loads TLS materials for the endpoint
func (c *EndpointMeta) WithTLSData(s store.Reader, contextName string) (Endpoint, error) {
	tlsData, err := LoadTLSData(s, contextName)
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
		authInfo.Token = c.TLSData.Token
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
	return kubernetes.NewKubernetesConfig(""), nil
}
