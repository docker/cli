package kubernetes

import (
	"io/ioutil"

	"github.com/docker/cli/cli/context"
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
	var ca, key, cert, token []byte
	if ca, err = readFileOrDefault(clientcfg.CAFile, clientcfg.CAData); err != nil {
		return Endpoint{}, err
	}
	if key, err = readFileOrDefault(clientcfg.KeyFile, clientcfg.KeyData); err != nil {
		return Endpoint{}, err
	}
	if cert, err = readFileOrDefault(clientcfg.CertFile, clientcfg.CertData); err != nil {
		return Endpoint{}, err
	}
	if token, err = readFileOrDefault(clientcfg.BearerTokenFile, []byte(clientcfg.BearerToken)); err != nil {
		return Endpoint{}, err
	}
	if len(token) == 0 {
		token = nil
	}
	var tlsData *TLSData
	if ca != nil || cert != nil || key != nil || token != nil {
		tlsData = &TLSData{
			TLSData: context.TLSData{
				CA:   ca,
				Cert: cert,
				Key:  key,
			},
			Token: string(token),
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
