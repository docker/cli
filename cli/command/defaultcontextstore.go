package command

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/context/store"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
)

const (
	// DefaultContextName is the name reserved for the default context (config & env based)
	DefaultContextName = "default"
)

// DefaultContext contains the default context data for all enpoints
type DefaultContext struct {
	Meta store.ContextMetadata
	TLS  store.ContextTLSData
}

// DefaultContextResolver is a function which resolves the default context base on the configuration and the env variables
type DefaultContextResolver func() (*DefaultContext, error)

// ContextStoreWithDefault implements the store.Store interface with a support for the default context
type ContextStoreWithDefault struct {
	store.Store
	Resolver DefaultContextResolver
}

// resolveDefaultContext creates a ContextMetadata for the current CLI invocation parameters
func resolveDefaultContext(opts *cliflags.CommonOptions, config *configfile.ConfigFile, stderr io.Writer) (*DefaultContext, error) {
	stackOrchestrator, err := GetStackOrchestrator("", "", config.StackOrchestrator, stderr)
	if err != nil {
		return nil, err
	}
	contextTLSData := store.ContextTLSData{
		Endpoints: make(map[string]store.EndpointTLSData),
	}
	contextMetadata := store.ContextMetadata{
		Endpoints: make(map[string]interface{}),
		Metadata: DockerContext{
			Description:       "",
			StackOrchestrator: stackOrchestrator,
		},
		Name: DefaultContextName,
	}

	dockerEP, err := resolveDefaultDockerEndpoint(opts)
	if err != nil {
		return nil, err
	}
	contextMetadata.Endpoints[docker.DockerEndpoint] = dockerEP.EndpointMeta
	if dockerEP.TLSData != nil {
		contextTLSData.Endpoints[docker.DockerEndpoint] = *dockerEP.TLSData.ToStoreTLSData()
	}

	// Default context uses env-based kubeconfig for Kubernetes endpoint configuration
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.Get(), ".kube/config")
	}
	kubeEP, err := kubernetes.FromKubeConfig(kubeconfig, "", "")
	if (stackOrchestrator == OrchestratorKubernetes || stackOrchestrator == OrchestratorAll) && err != nil {
		return nil, errors.Wrapf(err, "default orchestrator is %s but kubernetes endpoint could not be found", stackOrchestrator)
	}
	if err == nil {
		contextMetadata.Endpoints[kubernetes.KubernetesEndpoint] = kubeEP.EndpointMeta
		if kubeEP.TLSData != nil {
			contextTLSData.Endpoints[kubernetes.KubernetesEndpoint] = *kubeEP.TLSData.ToStoreTLSData()
		}
	}

	return &DefaultContext{Meta: contextMetadata, TLS: contextTLSData}, nil
}

// ListContexts implements store.Store's ListContexts
func (s *ContextStoreWithDefault) ListContexts() ([]store.ContextMetadata, error) {
	contextList, err := s.Store.ListContexts()
	if err != nil {
		return nil, err
	}
	defaultContext, err := s.Resolver()
	if err != nil {
		return nil, err
	}
	return append(contextList, defaultContext.Meta), nil
}

// CreateOrUpdateContext is not allowed for the default context and fails
func (s *ContextStoreWithDefault) CreateOrUpdateContext(meta store.ContextMetadata) error {
	if meta.Name == DefaultContextName {
		return errors.New("default context cannot be created nor updated")
	}
	return s.Store.CreateOrUpdateContext(meta)
}

// RemoveContext is not allowed for the default context and fails
func (s *ContextStoreWithDefault) RemoveContext(name string) error {
	if name == DefaultContextName {
		return errors.New("default context cannot be removed")
	}
	return s.Store.RemoveContext(name)
}

// GetContextMetadata implements store.Store's GetContextMetadata
func (s *ContextStoreWithDefault) GetContextMetadata(name string) (store.ContextMetadata, error) {
	if name == DefaultContextName {
		defaultContext, err := s.Resolver()
		if err != nil {
			return store.ContextMetadata{}, err
		}
		return defaultContext.Meta, nil
	}
	return s.Store.GetContextMetadata(name)
}

// ResetContextTLSMaterial is not implemented for default context and fails
func (s *ContextStoreWithDefault) ResetContextTLSMaterial(name string, data *store.ContextTLSData) error {
	if name == DefaultContextName {
		return errors.New("The default context store does not support ResetContextTLSMaterial")
	}
	return s.Store.ResetContextTLSMaterial(name, data)
}

// ResetContextEndpointTLSMaterial is not implemented for default context and fails
func (s *ContextStoreWithDefault) ResetContextEndpointTLSMaterial(contextName string, endpointName string, data *store.EndpointTLSData) error {
	if contextName == DefaultContextName {
		return errors.New("The default context store does not support ResetContextEndpointTLSMaterial")
	}
	return s.Store.ResetContextEndpointTLSMaterial(contextName, endpointName, data)
}

// ListContextTLSFiles implements store.Store's ListContextTLSFiles
func (s *ContextStoreWithDefault) ListContextTLSFiles(name string) (map[string]store.EndpointFiles, error) {
	if name == DefaultContextName {
		defaultContext, err := s.Resolver()
		if err != nil {
			return nil, err
		}
		tlsfiles := make(map[string]store.EndpointFiles)
		for epName, epTLSData := range defaultContext.TLS.Endpoints {
			var files store.EndpointFiles
			for filename := range epTLSData.Files {
				files = append(files, filename)
			}
			tlsfiles[epName] = files
		}
		return tlsfiles, nil
	}
	return s.Store.ListContextTLSFiles(name)
}

// GetContextTLSData implements store.Store's GetContextTLSData
func (s *ContextStoreWithDefault) GetContextTLSData(contextName, endpointName, fileName string) ([]byte, error) {
	if contextName == DefaultContextName {
		defaultContext, err := s.Resolver()
		if err != nil {
			return nil, err
		}
		if defaultContext.TLS.Endpoints[endpointName].Files[fileName] == nil {
			return nil, &noDefaultTLSDataError{endpointName: endpointName, fileName: fileName}
		}
		return defaultContext.TLS.Endpoints[endpointName].Files[fileName], nil

	}
	return s.Store.GetContextTLSData(contextName, endpointName, fileName)
}

type noDefaultTLSDataError struct {
	endpointName string
	fileName     string
}

func (e *noDefaultTLSDataError) Error() string {
	return fmt.Sprintf("tls data for %s/%s/%s does not exist", DefaultContextName, e.endpointName, e.fileName)
}

// NotFound satisfies interface github.com/docker/docker/errdefs.ErrNotFound
func (e *noDefaultTLSDataError) NotFound() {}

// IsTLSDataDoesNotExist satisfies github.com/docker/cli/cli/context/store.tlsDataDoesNotExist
func (e *noDefaultTLSDataError) IsTLSDataDoesNotExist() {}

// GetContextStorageInfo implements store.Store's GetContextStorageInfo
func (s *ContextStoreWithDefault) GetContextStorageInfo(contextName string) store.ContextStorageInfo {
	if contextName == DefaultContextName {
		return store.ContextStorageInfo{MetadataPath: "<IN MEMORY>", TLSPath: "<IN MEMORY>"}
	}
	return s.Store.GetContextStorageInfo(contextName)
}
