package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
)

const (
	keyFrom          = "from"
	keyHost          = "host"
	keyCA            = "ca"
	keyCert          = "cert"
	keyKey           = "key"
	keySkipTLSVerify = "skip-tls-verify"
	keyKubeconfig    = "config-file"
	keyKubecontext   = "context-override"
	keyKubenamespace = "namespace-override"
)

type configKeyDescription struct {
	name        string
	description string
}

var (
	allowedDockerConfigKeys = map[string]struct{}{
		keyFrom:          {},
		keyHost:          {},
		keyCA:            {},
		keyCert:          {},
		keyKey:           {},
		keySkipTLSVerify: {},
	}
	allowedKubernetesConfigKeys = map[string]struct{}{
		keyFrom:          {},
		keyKubeconfig:    {},
		keyKubecontext:   {},
		keyKubenamespace: {},
	}
	dockerConfigKeysDescriptions = []configKeyDescription{
		{
			name:        keyFrom,
			description: "Copy named context's Docker endpoint configuration",
		},
		{
			name:        keyHost,
			description: "Docker endpoint on which to connect",
		},
		{
			name:        keyCA,
			description: "Trust certs signed only by this CA",
		},
		{
			name:        keyCert,
			description: "Path to TLS certificate file",
		},
		{
			name:        keyKey,
			description: "Path to TLS key file",
		},
		{
			name:        keySkipTLSVerify,
			description: "Skip TLS certificate validation",
		},
	}
	kubernetesConfigKeysDescriptions = []configKeyDescription{
		{
			name:        keyFrom,
			description: "Copy named context's Kubernetes endpoint configuration",
		},
		{
			name:        keyKubeconfig,
			description: "Path to a Kubernetes config file",
		},
		{
			name:        keyKubecontext,
			description: "Overrides the context set in the kubernetes config file",
		},
		{
			name:        keyKubenamespace,
			description: "Overrides the namespace set in the kubernetes config file",
		},
	}
)

func parseBool(config map[string]string, name string) (bool, error) {
	strVal, ok := config[name]
	if !ok {
		return false, nil
	}
	res, err := strconv.ParseBool(strVal)
	return res, errors.Wrap(err, name)
}

func validateConfig(config map[string]string, allowedKeys map[string]struct{}) error {
	var errs []string
	for k := range config {
		if _, ok := allowedKeys[k]; !ok {
			errs = append(errs, fmt.Sprintf("%s: unrecognized config key", k))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}

func getDockerEndpoint(dockerCli command.Cli, config map[string]string) (docker.Endpoint, error) {
	if err := validateConfig(config, allowedDockerConfigKeys); err != nil {
		return docker.Endpoint{}, err
	}
	if contextName, ok := config[keyFrom]; ok {
		metadata, err := dockerCli.ContextStore().GetContextMetadata(contextName)
		if err != nil {
			return docker.Endpoint{}, err
		}
		if ep, ok := metadata.Endpoints[docker.DockerEndpoint].(docker.EndpointMeta); ok {
			return docker.Endpoint{EndpointMeta: ep}, nil
		}
		return docker.Endpoint{}, errors.Errorf("unable to get endpoint from context %q", contextName)
	}
	tlsData, err := context.TLSDataFromFiles(config[keyCA], config[keyCert], config[keyKey])
	if err != nil {
		return docker.Endpoint{}, err
	}
	skipTLSVerify, err := parseBool(config, keySkipTLSVerify)
	if err != nil {
		return docker.Endpoint{}, err
	}
	ep := docker.Endpoint{
		EndpointMeta: docker.EndpointMeta{
			Host:          config[keyHost],
			SkipTLSVerify: skipTLSVerify,
		},
		TLSData: tlsData,
	}
	// try to resolve a docker client, validating the configuration
	opts, err := ep.ClientOpts()
	if err != nil {
		return docker.Endpoint{}, errors.Wrap(err, "invalid docker endpoint options")
	}
	if _, err := client.NewClientWithOpts(opts...); err != nil {
		return docker.Endpoint{}, errors.Wrap(err, "unable to apply docker endpoint options")
	}
	return ep, nil
}

func getDockerEndpointMetadataAndTLS(dockerCli command.Cli, config map[string]string) (docker.EndpointMeta, *store.EndpointTLSData, error) {
	ep, err := getDockerEndpoint(dockerCli, config)
	if err != nil {
		return docker.EndpointMeta{}, nil, err
	}
	return ep.EndpointMeta, ep.TLSData.ToStoreTLSData(), nil
}

func getKubernetesEndpoint(dockerCli command.Cli, config map[string]string) (*kubernetes.Endpoint, error) {
	if err := validateConfig(config, allowedKubernetesConfigKeys); err != nil {
		return nil, err
	}
	if len(config) == 0 {
		return nil, nil
	}
	if contextName, ok := config[keyFrom]; ok {
		ctxMeta, err := dockerCli.ContextStore().GetContextMetadata(contextName)
		if err != nil {
			return nil, err
		}
		endpointMeta := kubernetes.EndpointFromContext(ctxMeta)
		if endpointMeta != nil {
			res, err := endpointMeta.WithTLSData(dockerCli.ContextStore(), dockerCli.CurrentContext())
			if err != nil {
				return nil, err
			}
			return &res, nil
		}

		// fallback to env-based kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = filepath.Join(homedir.Get(), ".kube/config")
		}
		ep, err := kubernetes.FromKubeConfig(kubeconfig, "", "")
		if err != nil {
			return nil, err
		}
		return &ep, nil
	}
	if config[keyKubeconfig] != "" {
		ep, err := kubernetes.FromKubeConfig(config[keyKubeconfig], config[keyKubecontext], config[keyKubenamespace])
		if err != nil {
			return nil, err
		}
		return &ep, nil
	}
	return nil, nil
}

func getKubernetesEndpointMetadataAndTLS(dockerCli command.Cli, config map[string]string) (*kubernetes.EndpointMeta, *store.EndpointTLSData, error) {
	ep, err := getKubernetesEndpoint(dockerCli, config)
	if err != nil {
		return nil, nil, err
	}
	if ep == nil {
		return nil, nil, err
	}
	return &ep.EndpointMeta, ep.TLSData.ToStoreTLSData(), nil
}
