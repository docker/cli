package store

import (
	"archive/tar"
	_ "crypto/sha256" // ensure ids can be computed
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/docker/errdefs"
	digest "github.com/opencontainers/go-digest"
)

// Store provides a context store for easily remembering endpoints configuration
type Store interface {
	Reader
	Lister
	Writer
	StorageInfoProvider
}

// Reader provides read-only (without list) access to context data
type Reader interface {
	GetMetadata(name string) (Metadata, error)
	ListTLSFiles(name string) (map[string]EndpointFiles, error)
	GetTLSData(contextName, endpointName, fileName string) ([]byte, error)
}

// Lister provides listing of contexts
type Lister interface {
	List() ([]Metadata, error)
}

// ReaderLister combines Reader and Lister interfaces
type ReaderLister interface {
	Reader
	Lister
}

// StorageInfoProvider provides more information about storage details of contexts
type StorageInfoProvider interface {
	GetStorageInfo(contextName string) StorageInfo
}

// Writer provides write access to context data
type Writer interface {
	CreateOrUpdate(meta Metadata) error
	Remove(name string) error
	ResetTLSMaterial(name string, data *ContextTLSData) error
	ResetEndpointTLSMaterial(contextName string, endpointName string, data *EndpointTLSData) error
}

// ReaderWriter combines Reader and Writer interfaces
type ReaderWriter interface {
	Reader
	Writer
}

// Metadata contains metadata about a context and its endpoints
type Metadata struct {
	Name      string                 `json:",omitempty"`
	Metadata  interface{}            `json:",omitempty"`
	Endpoints map[string]interface{} `json:",omitempty"`
}

// StorageInfo contains data about where a given context is stored
type StorageInfo struct {
	MetadataPath string
	TLSPath      string
}

// EndpointTLSData represents tls data for a given endpoint
type EndpointTLSData struct {
	Files map[string][]byte
}

// ContextTLSData represents tls data for a whole context
type ContextTLSData struct {
	Endpoints map[string]EndpointTLSData
}

// New creates a store from a given directory.
// If the directory does not exist or is empty, initialize it
func New(dir string, cfg Config) Store {
	metaRoot := filepath.Join(dir, metadataDir)
	tlsRoot := filepath.Join(dir, tlsDir)

	return &store{
		meta: &metadataStore{
			root:   metaRoot,
			config: cfg,
		},
		tls: &tlsStore{
			root: tlsRoot,
		},
	}
}

type store struct {
	meta *metadataStore
	tls  *tlsStore
}

func (s *store) List() ([]Metadata, error) {
	return s.meta.list()
}

func (s *store) CreateOrUpdate(meta Metadata) error {
	return s.meta.createOrUpdate(meta)
}

func (s *store) Remove(name string) error {
	id := contextdirOf(name)
	if err := s.meta.remove(id); err != nil {
		return patchErrContextName(err, name)
	}
	return patchErrContextName(s.tls.removeAllContextData(id), name)
}

func (s *store) GetMetadata(name string) (Metadata, error) {
	res, err := s.meta.get(contextdirOf(name))
	patchErrContextName(err, name)
	return res, err
}

func (s *store) ResetTLSMaterial(name string, data *ContextTLSData) error {
	id := contextdirOf(name)
	if err := s.tls.removeAllContextData(id); err != nil {
		return patchErrContextName(err, name)
	}
	if data == nil {
		return nil
	}
	for ep, files := range data.Endpoints {
		for fileName, data := range files.Files {
			if err := s.tls.createOrUpdate(id, ep, fileName, data); err != nil {
				return patchErrContextName(err, name)
			}
		}
	}
	return nil
}

func (s *store) ResetEndpointTLSMaterial(contextName string, endpointName string, data *EndpointTLSData) error {
	id := contextdirOf(contextName)
	if err := s.tls.removeAllEndpointData(id, endpointName); err != nil {
		return patchErrContextName(err, contextName)
	}
	if data == nil {
		return nil
	}
	for fileName, data := range data.Files {
		if err := s.tls.createOrUpdate(id, endpointName, fileName, data); err != nil {
			return patchErrContextName(err, contextName)
		}
	}
	return nil
}

func (s *store) ListTLSFiles(name string) (map[string]EndpointFiles, error) {
	res, err := s.tls.listContextData(contextdirOf(name))
	return res, patchErrContextName(err, name)
}

func (s *store) GetTLSData(contextName, endpointName, fileName string) ([]byte, error) {
	res, err := s.tls.getData(contextdirOf(contextName), endpointName, fileName)
	return res, patchErrContextName(err, contextName)
}

func (s *store) GetStorageInfo(contextName string) StorageInfo {
	dir := contextdirOf(contextName)
	return StorageInfo{
		MetadataPath: s.meta.contextDir(dir),
		TLSPath:      s.tls.contextDir(dir),
	}
}

// Export exports an existing namespace into an opaque data stream
// This stream is actually a tarball containing context metadata and TLS materials, but it does
// not map 1:1 the layout of the context store (don't try to restore it manually without calling store.Import)
func Export(name string, s Reader) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		tw := tar.NewWriter(writer)
		defer tw.Close()
		defer writer.Close()
		meta, err := s.GetMetadata(name)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		metaBytes, err := json.Marshal(&meta)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		if err = tw.WriteHeader(&tar.Header{
			Name: metaFile,
			Mode: 0644,
			Size: int64(len(metaBytes)),
		}); err != nil {
			writer.CloseWithError(err)
			return
		}
		if _, err = tw.Write(metaBytes); err != nil {
			writer.CloseWithError(err)
			return
		}
		tlsFiles, err := s.ListTLSFiles(name)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		if err = tw.WriteHeader(&tar.Header{
			Name:     "tls",
			Mode:     0700,
			Size:     0,
			Typeflag: tar.TypeDir,
		}); err != nil {
			writer.CloseWithError(err)
			return
		}
		for endpointName, endpointFiles := range tlsFiles {
			if err = tw.WriteHeader(&tar.Header{
				Name:     path.Join("tls", endpointName),
				Mode:     0700,
				Size:     0,
				Typeflag: tar.TypeDir,
			}); err != nil {
				writer.CloseWithError(err)
				return
			}
			for _, fileName := range endpointFiles {
				data, err := s.GetTLSData(name, endpointName, fileName)
				if err != nil {
					writer.CloseWithError(err)
					return
				}
				if err = tw.WriteHeader(&tar.Header{
					Name: path.Join("tls", endpointName, fileName),
					Mode: 0600,
					Size: int64(len(data)),
				}); err != nil {
					writer.CloseWithError(err)
					return
				}
				if _, err = tw.Write(data); err != nil {
					writer.CloseWithError(err)
					return
				}
			}
		}
	}()
	return reader
}

// Import imports an exported context into a store
func Import(name string, s Writer, reader io.Reader) error {
	tr := tar.NewReader(reader)
	tlsData := ContextTLSData{
		Endpoints: map[string]EndpointTLSData{},
	}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag == tar.TypeDir {
			// skip this entry, only taking files into account
			continue
		}
		if hdr.Name == metaFile {
			data, err := ioutil.ReadAll(tr)
			if err != nil {
				return err
			}
			var meta Metadata
			if err := json.Unmarshal(data, &meta); err != nil {
				return err
			}
			meta.Name = name
			if err := s.CreateOrUpdate(meta); err != nil {
				return err
			}
		} else if strings.HasPrefix(hdr.Name, "tls/") {
			relative := strings.TrimPrefix(hdr.Name, "tls/")
			parts := strings.SplitN(relative, "/", 2)
			if len(parts) != 2 {
				return errors.New("archive format is invalid")
			}
			endpointName := parts[0]
			fileName := parts[1]
			data, err := ioutil.ReadAll(tr)
			if err != nil {
				return err
			}
			if _, ok := tlsData.Endpoints[endpointName]; !ok {
				tlsData.Endpoints[endpointName] = EndpointTLSData{
					Files: map[string][]byte{},
				}
			}
			tlsData.Endpoints[endpointName].Files[fileName] = data
		}
	}
	return s.ResetTLSMaterial(name, &tlsData)
}

type setContextName interface {
	setContext(name string)
}

type contextDoesNotExistError struct {
	name string
}

func (e *contextDoesNotExistError) Error() string {
	return fmt.Sprintf("context %q does not exist", e.name)
}

func (e *contextDoesNotExistError) setContext(name string) {
	e.name = name
}

// NotFound satisfies interface github.com/docker/docker/errdefs.ErrNotFound
func (e *contextDoesNotExistError) NotFound() {}

type tlsDataDoesNotExist interface {
	errdefs.ErrNotFound
	IsTLSDataDoesNotExist()
}

type tlsDataDoesNotExistError struct {
	context, endpoint, file string
}

func (e *tlsDataDoesNotExistError) Error() string {
	return fmt.Sprintf("tls data for %s/%s/%s does not exist", e.context, e.endpoint, e.file)
}

func (e *tlsDataDoesNotExistError) setContext(name string) {
	e.context = name
}

// NotFound satisfies interface github.com/docker/docker/errdefs.ErrNotFound
func (e *tlsDataDoesNotExistError) NotFound() {}

// IsTLSDataDoesNotExist satisfies tlsDataDoesNotExist
func (e *tlsDataDoesNotExistError) IsTLSDataDoesNotExist() {}

// IsErrContextDoesNotExist checks if the given error is a "context does not exist" condition
func IsErrContextDoesNotExist(err error) bool {
	_, ok := err.(*contextDoesNotExistError)
	return ok
}

// IsErrTLSDataDoesNotExist checks if the given error is a "context does not exist" condition
func IsErrTLSDataDoesNotExist(err error) bool {
	_, ok := err.(tlsDataDoesNotExist)
	return ok
}

type contextdir string

func contextdirOf(name string) contextdir {
	return contextdir(digest.FromString(name).Encoded())
}

func patchErrContextName(err error, name string) error {
	if typed, ok := err.(setContextName); ok {
		typed.setContext(name)
	}
	return err
}
