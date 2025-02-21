package context // import "docker.com/cli/v28/cli/command/context"

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/docker/cli/v28/cli/context"
	"github.com/docker/cli/v28/cli/context/docker"
	"github.com/docker/cli/v28/cli/context/store"
	"github.com/docker/docker/client"
)

const (
	keyFrom          = "from"
	keyHost          = "host"
	keyCA            = "ca"
	keyCert          = "cert"
	keyKey           = "key"
	keySkipTLSVerify = "skip-tls-verify"
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
)

func parseBool(config map[string]string, name string) (bool, error) {
	strVal, ok := config[name]
	if !ok {
		return false, nil
	}
	res, err := strconv.ParseBool(strVal)
	if err != nil {
		var nErr *strconv.NumError
		if errors.As(err, &nErr) {
			return res, fmt.Errorf("%s: parsing %q: %w", name, nErr.Num, nErr.Err)
		}
		return res, fmt.Errorf("%s: %w", name, err)
	}
	return res, nil
}

func validateConfig(config map[string]string, allowedKeys map[string]struct{}) error {
	var errs []error
	for k := range config {
		if _, ok := allowedKeys[k]; !ok {
			errs = append(errs, errors.New("unrecognized config key: "+k))
		}
	}
	return errors.Join(errs...)
}

func getDockerEndpoint(contextStore store.Reader, config map[string]string) (docker.Endpoint, error) {
	if err := validateConfig(config, allowedDockerConfigKeys); err != nil {
		return docker.Endpoint{}, err
	}
	if contextName, ok := config[keyFrom]; ok {
		metadata, err := contextStore.GetMetadata(contextName)
		if err != nil {
			return docker.Endpoint{}, err
		}
		if ep, ok := metadata.Endpoints[docker.DockerEndpoint].(docker.EndpointMeta); ok {
			return docker.Endpoint{EndpointMeta: ep}, nil
		}
		return docker.Endpoint{}, fmt.Errorf("unable to get endpoint from context %q", contextName)
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
		return docker.Endpoint{}, fmt.Errorf("invalid docker endpoint options: %w", err)
	}
	// FIXME(thaJeztah): this creates a new client (but discards it) only to validate the options; are the validation steps above not enough?
	if _, err := client.NewClientWithOpts(opts...); err != nil {
		return docker.Endpoint{}, fmt.Errorf("unable to apply docker endpoint options: %w", err)
	}
	return ep, nil
}

func getDockerEndpointMetadataAndTLS(contextStore store.Reader, config map[string]string) (docker.EndpointMeta, *store.EndpointTLSData, error) {
	ep, err := getDockerEndpoint(contextStore, config)
	if err != nil {
		return docker.EndpointMeta{}, nil, err
	}
	return ep.EndpointMeta, ep.TLSData.ToStoreTLSData(), nil
}
