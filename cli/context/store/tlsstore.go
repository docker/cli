package store

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

const tlsDir = "tls"

type tlsStore struct {
	root string
}

func (s *tlsStore) contextDir(name string) string {
	return filepath.Join(s.root, name)
}

func (s *tlsStore) endpointDir(contextName, name string) string {
	return filepath.Join(s.root, contextName, name)
}

func (s *tlsStore) filePath(contextName, endpointName, filename string) string {
	return filepath.Join(s.root, contextName, endpointName, filename)
}

func (s *tlsStore) createOrUpdate(contextName, endpointName, filename string, data []byte) error {
	epdir := s.endpointDir(contextName, endpointName)
	parentOfRoot := filepath.Dir(s.root)
	if err := os.MkdirAll(parentOfRoot, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(epdir, 0700); err != nil {
		return err
	}
	return ioutil.WriteFile(s.filePath(contextName, endpointName, filename), data, 0600)
}

func (s *tlsStore) getData(contextName, endpointName, filename string) ([]byte, error) {
	data, err := ioutil.ReadFile(s.filePath(contextName, endpointName, filename))
	if err != nil {
		return nil, convertTLSDataDoesNotExist(contextName, endpointName, filename, err)
	}
	return data, nil
}

func (s *tlsStore) remove(contextName, endpointName, filename string) error {
	err := os.Remove(s.filePath(contextName, endpointName, filename))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *tlsStore) removeAllEndpointData(contextName, endpointName string) error {
	return os.RemoveAll(s.endpointDir(contextName, endpointName))
}

func (s *tlsStore) removeAllContextData(contextName string) error {
	return os.RemoveAll(s.contextDir(contextName))
}

func (s *tlsStore) listContextData(contextName string) (map[string]EndpointFiles, error) {
	epFSs, err := ioutil.ReadDir(s.contextDir(contextName))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]EndpointFiles{}, nil
		}
		return nil, err
	}
	r := make(map[string]EndpointFiles)
	for _, epFS := range epFSs {
		if epFS.IsDir() {
			epDir := s.endpointDir(contextName, epFS.Name())
			fss, err := ioutil.ReadDir(epDir)
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

func convertTLSDataDoesNotExist(context, endpoint, file string, err error) error {
	if os.IsNotExist(err) {
		return &tlsDataDoesNotExistError{context: context, endpoint: endpoint, file: file}
	}
	return err
}
