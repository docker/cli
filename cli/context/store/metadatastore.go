package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
)

const (
	metadataDir = "meta"
	metaFile    = "meta.json"
)

type metadataStore struct {
	root   string
	config Config
}

func (s *metadataStore) contextDir(name string) string {
	return filepath.Join(s.root, name)
}

func (s *metadataStore) createOrUpdate(name string, meta ContextMetadata) error {
	contextDir := s.contextDir(name)
	if err := os.MkdirAll(contextDir, 0755); err != nil {
		return err
	}
	bytes, err := json.Marshal(&meta)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(contextDir, metaFile), bytes, 0644)
}

func parseTypedOrMap(payload []byte, getter TypeGetter) (interface{}, error) {
	if len(payload) == 0 || string(payload) == "null" {
		return nil, nil
	}
	if getter == nil {
		var res map[string]interface{}
		if err := json.Unmarshal(payload, &res); err != nil {
			return nil, err
		}
		return res, nil
	}
	typed := getter()
	if err := json.Unmarshal(payload, typed); err != nil {
		return nil, err
	}
	return reflect.ValueOf(typed).Elem().Interface(), nil
}

func (s *metadataStore) get(name string) (ContextMetadata, error) {
	contextDir := s.contextDir(name)
	bytes, err := ioutil.ReadFile(filepath.Join(contextDir, metaFile))
	if err != nil {
		return ContextMetadata{}, convertContextDoesNotExist(name, err)
	}
	var untyped untypedContextMetadata
	r := ContextMetadata{
		Endpoints: make(map[string]interface{}),
	}
	if err := json.Unmarshal(bytes, &untyped); err != nil {
		return ContextMetadata{}, err
	}
	if r.Metadata, err = parseTypedOrMap(untyped.Metadata, s.config.contextType); err != nil {
		return ContextMetadata{}, err
	}
	for k, v := range untyped.Endpoints {
		if r.Endpoints[k], err = parseTypedOrMap(v, s.config.endpointTypes[k]); err != nil {
			return ContextMetadata{}, err
		}
	}
	return r, err
}

func (s *metadataStore) remove(name string) error {
	contextDir := s.contextDir(name)
	return os.RemoveAll(contextDir)
}

func (s *metadataStore) list() (map[string]ContextMetadata, error) {
	ctxNames, err := listRecursivelyMetadataDirs(s.root)
	if err != nil {
		if os.IsNotExist(err) {
			// store is empty, meta dir does not exist yet
			// this should not be considered an error
			return map[string]ContextMetadata{}, nil
		}
		return nil, err
	}
	res := make(map[string]ContextMetadata)
	for _, name := range ctxNames {
		res[name], err = s.get(name)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func isContextDir(path string) bool {
	s, err := os.Stat(filepath.Join(path, metaFile))
	if err != nil {
		return false
	}
	return !s.IsDir()
}

func listRecursivelyMetadataDirs(root string) ([]string, error) {
	fis, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, fi := range fis {
		if fi.IsDir() {
			if isContextDir(filepath.Join(root, fi.Name())) {
				result = append(result, fi.Name())
			}
			subs, err := listRecursivelyMetadataDirs(filepath.Join(root, fi.Name()))
			if err != nil {
				return nil, err
			}
			for _, s := range subs {
				result = append(result, fmt.Sprintf("%s/%s", fi.Name(), s))
			}
		}
	}
	return result, nil
}

func convertContextDoesNotExist(name string, err error) error {
	if os.IsNotExist(err) {
		return &contextDoesNotExistError{name: name}
	}
	return err
}

type untypedContextMetadata struct {
	Metadata  json.RawMessage            `json:"metadata,omitempty"`
	Endpoints map[string]json.RawMessage `json:"endpoints,omitempty"`
}
