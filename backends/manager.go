package backends

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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

type BackendMetadata struct {
	Name    string
	Version string
}

const (
	backendPrefix = "docker-"
	backendSuffix = "-backend"
)

// ListBackends produces a list of available backends on the system and their context types
func ListBackends() ([]Backend, error) {
	return listBackendsFrom(getDockerCliBackendDir(), fakeMetadata)
}

// GetBackend get backend for a given context type
func GetBackend(contextType string) (*Backend, error) {
	backends, err := ListBackends()
	if err != nil {
		return nil, err
	}
	for _, backend := range backends {
		if utils.StrSliceContains(backend.SupportedTypes, contextType) {
			return &backend, nil
		}
	}
	return nil, fmt.Errorf("no available backend for context type %q", contextType)
}

type extractMetadataFunc func(binary string) (BackendMetadata, error)

func fakeMetadata(binary string) (BackendMetadata, error) {
	name := strings.TrimPrefix(filepath.Base(binary), backendPrefix)
	return BackendMetadata{Name: strings.TrimSuffix(name, ".exe"), Version: "1.0"}, nil
}

func listBackendsFrom(backendDir string, extractVersion extractMetadataFunc) ([]Backend, error) {
	if fi, err := os.Stat(backendDir); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("%q is not a directory, unable to list backends", backendDir)
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
		fileName := candidate.Name()
		name := strings.TrimSuffix(fileName, ".exe")
		if !strings.HasPrefix(name, backendPrefix) || !strings.HasSuffix(name, backendSuffix) {
			continue
		}
		contextTypes := getContextTypes(name)
		path := filepath.Join(backendDir, fileName)
		metadata, err := extractVersion(path)
		if err != nil {
			continue
		}
		result = append(result, Backend{Name: metadata.Name, Path: path, Version: metadata.Version, SupportedTypes: contextTypes})
	}
	return result, nil
}

func getContextTypes(name string) []string {
	name = strings.TrimPrefix(name, backendPrefix)
	return strings.Split(strings.TrimSuffix(name, backendSuffix), "-")
}
