package store

import (
	"os"
	"path/filepath"
)

const tlsDir = "tls"

type tlsStore struct {
	root string
}

func (s *tlsStore) contextDir(name string) string {
	return filepath.Join(s.root, string(contextdirOf(name)))
}

func (s *tlsStore) endpointDir(name, endpointName string) string {
	return filepath.Join(s.contextDir(name), endpointName)
}

func (s *tlsStore) createOrUpdate(name, endpointName, filename string, data []byte) error {
	parentOfRoot := filepath.Dir(s.root)
	if err := os.MkdirAll(parentOfRoot, 0755); err != nil {
		return err
	}
	endpointDir := s.endpointDir(name, endpointName)
	if err := os.MkdirAll(endpointDir, 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(endpointDir, filename), data, 0600)
}

func (s *tlsStore) getData(name, endpointName, filename string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(s.endpointDir(name, endpointName), filename))
	if err != nil {
		return nil, convertTLSDataDoesNotExist(endpointName, filename, err)
	}
	return data, nil
}

// remove removes a TLS data from an endpoint
// TODO(thaJeztah) tlsStore.remove() is not used anywhere outside of tests; should we use removeAllEndpointData() only?
func (s *tlsStore) remove(name, endpointName, filename string) error {
	err := os.Remove(filepath.Join(s.endpointDir(name, endpointName), filename))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *tlsStore) removeAllEndpointData(name, endpointName string) error {
	return os.RemoveAll(s.endpointDir(name, endpointName))
}

func (s *tlsStore) removeAllContextData(name string) error {
	return os.RemoveAll(s.contextDir(name))
}

func (s *tlsStore) listContextData(name string) (map[string]EndpointFiles, error) {
	contextDir := s.contextDir(name)
	epFSs, err := os.ReadDir(contextDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]EndpointFiles{}, nil
		}
		return nil, err
	}
	r := make(map[string]EndpointFiles)
	for _, epFS := range epFSs {
		if epFS.IsDir() {
			fss, err := os.ReadDir(filepath.Join(contextDir, epFS.Name()))
			if err != nil {
				return nil, err
			}
			var files EndpointFiles
			for _, fs := range fss {
				if !fs.IsDir() {
					files = append(files, fs.Name())
				}
			}
			r[epFS.Name()] = files
		}
	}
	return r, nil
}

// EndpointFiles is a slice of strings representing file names
type EndpointFiles []string

func convertTLSDataDoesNotExist(endpoint, file string, err error) error {
	if os.IsNotExist(err) {
		return &tlsDataDoesNotExistError{endpoint: endpoint, file: file}
	}
	return err
}
