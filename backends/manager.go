package backends

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/theupdateframework/notary/tuf/utils"
)

// Backend info for available backends
type Backend struct {
	Name           string   `json:",omitempty"`
	Path           string   `json:"-"`
	Version        string   `json:",omitempty"`
	SupportedTypes []string `json:"-"`
	Err            error    `json:"-"`
}

// BackendMetadata backend metadata
type BackendMetadata struct {
	Name    string `json:",omitempty"`
	Version string `json:",omitempty"`
}

const (
	backendPrefix = "docker-"
	backendSuffix = "-backend"
)

var allowedBackends = map[string][]string{"docker-compose-cli-backend": {"aci", "ecs", "local"}}

// ListBackends produces a list of available backends on the system and their context types
func ListBackends() []Backend {
	backends, err := listBackendsFrom(getDockerCliBackendDir(), getMetadata, allowedBackends)
	if err != nil {
		logrus.Debug("Skippping backend due to: " + err.Error())
		return nil
	}
	return backends
}

func getMetadata(binary string) (BackendMetadata, error) {
	output, err := shellout(binary, "backend-metadata")
	if err != nil {
		return BackendMetadata{}, err
	}
	metadata := BackendMetadata{}
	if err := json.Unmarshal(output, &metadata); err != nil {
		return BackendMetadata{}, err
	}
	return metadata, nil
}

// GetBackend get backend for a given context type
func GetBackend(contextType string) (*Backend, error) {
	backends := ListBackends()
	for _, backend := range backends {
		if utils.StrSliceContains(backend.SupportedTypes, contextType) {
			return &backend, nil
		}
	}
	return nil, fmt.Errorf("no available backend for context type %q", contextType)
}

type extractMetadataFunc func(binary string) (BackendMetadata, error)

func listBackendsFrom(backendDir string, extractMetadata extractMetadataFunc, allowedBackendFiles map[string][]string) ([]Backend, error) {
	fi, err := os.Stat(backendDir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("backend folder %q is not a folder, skipping CLI backends", backendDir)
	}
	dentries, err := ioutil.ReadDir(backendDir)
	if err != nil {
		return nil, err
	}
	result := []Backend{}
	for _, candidate := range dentries {
		switch candidate.Mode() & os.ModeType {
		case 0, os.ModeSymlink:
			// Regular file or symlink, keep going
		default:
			// Something else, ignore.
			continue
		}
		filename := candidate.Name()
		path := filepath.Join(backendDir, filename)
		withoutExe := strings.TrimSuffix(filename, ".exe")
		if !strings.HasPrefix(withoutExe, backendPrefix) || !strings.HasSuffix(withoutExe, backendSuffix) {
			continue
		}
		contextTypes, allowed := allowedBackendFiles[withoutExe]
		if !allowed {
			logrus.Debugf("Invalid backend : backend binary %q is not allowed", filename)
			continue
		}

		metadata, err := extractMetadata(path)
		if err != nil {
			continue
		}
		result = append(result, Backend{Name: metadata.Name, Path: path, Version: metadata.Version, SupportedTypes: contextTypes})
	}
	return result, nil
}
