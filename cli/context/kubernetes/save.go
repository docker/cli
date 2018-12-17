package kubernetes

import (
	"io/ioutil"

	"github.com/docker/cli/cli/context"
	"github.com/docker/cli/cli/context/store"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// FromKubeConfig creates a Kubernetes endpoint from a Kubeconfig file
func FromKubeConfig(kubeconfig, kubeContext, namespaceOverride string) (Endpoint, error) {
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: kubeContext, Context: clientcmdapi.Context{Namespace: namespaceOverride}})
	ns, _, err := cfg.Namespace()
	if err != nil {
		return Endpoint{}, err
	}
	clientcfg, err := cfg.ClientConfig()
	if err != nil {
		return Endpoint{}, err
	}
	var ca, key, cert []byte
	if ca, err = readFileOrDefault(clientcfg.CAFile, clientcfg.CAData); err != nil {
		return Endpoint{}, err
	}
	if key, err = readFileOrDefault(clientcfg.KeyFile, clientcfg.KeyData); err != nil {
		return Endpoint{}, err
	}
	if cert, err = readFileOrDefault(clientcfg.CertFile, clientcfg.CertData); err != nil {
		return Endpoint{}, err
	}
	var tlsData *context.TLSData
	if ca != nil || cert != nil || key != nil {
		tlsData = &context.TLSData{
			CA:   ca,
			Cert: cert,
			Key:  key,
		}
	}
	return Endpoint{
		EndpointMeta: EndpointMeta{
			EndpointMetaBase: context.EndpointMetaBase{
				Host:          clientcfg.Host,
				SkipTLSVerify: clientcfg.Insecure,
			},
			DefaultNamespace: ns,
			AuthProvider:     clientcfg.AuthProvider,
			Exec:             clientcfg.ExecProvider,
		},
		TLSData: tlsData,
	}, nil
}

func readFileOrDefault(path string, defaultValue []byte) ([]byte, error) {
	if path != "" {
		return ioutil.ReadFile(path)
	}
	return defaultValue, nil
}

// Save the endpoint metadata and TLS bundle in the context store
func (ep *Endpoint) Save(s store.Store, contextName string) error {
	tlsData := ep.TLSData.ToStoreTLSData()
	existingContext, err := s.GetContextMetadata(contextName)
	if err != nil && !store.IsErrContextDoesNotExist(err) {
		return err
	}
	if existingContext.Endpoints == nil {
		existingContext.Endpoints = make(map[string]interface{})
	}
	existingContext.Endpoints[KubernetesEndpoint] = ep.EndpointMeta
	if err := s.CreateOrUpdateContext(contextName, existingContext); err != nil {
		return err
	}
	return s.ResetContextEndpointTLSMaterial(contextName, KubernetesEndpoint, tlsData)
}
