package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli/manifest/types"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

// Store manages local storage of image distribution manifests
type Store interface {
	Remove(listRef reference.Reference) error
	Get(listRef reference.Reference, manifest reference.Reference) ([]types.ImageManifest, error)
	GetList(listRef reference.Reference) ([]types.ImageManifest, error)
	Save(listRef reference.Reference, manifest reference.Reference, image ...types.ImageManifest) error
}

// fsStore manages manifest files stored on the local filesystem
type fsStore struct {
	root string
}

// NewStore returns a new store for a local file path
func NewStore(root string) Store {
	return &fsStore{root: root}
}

// Remove a manifest list from local storage
func (s *fsStore) Remove(listRef reference.Reference) error {
	path := filepath.Join(s.root, makeFilesafeName(listRef.String()))
	return os.RemoveAll(path)
}

// Get returns the local manifest
func (s *fsStore) Get(listRef reference.Reference, manifest reference.Reference) ([]types.ImageManifest, error) {
	var imgs []types.ImageManifest
	filename := manifestToFilename(s.root, listRef.String(), manifest.String())

	img, err := s.getFromFilename(manifest, filename)
	if err != nil {
		return nil, err
	}
	imgs = append(imgs, img)

	i := 2
	for {
		img, err := s.getFromFilename(manifest, fmt.Sprintf("%s_%d", filename, i))
		if errors.Is(err, types.ErrManifestNotFound) {
			break
		} else if err != nil {
			return nil, err
		}
		imgs = append(imgs, img)

		i++
	}

	return imgs, nil
}

func (s *fsStore) getFromFilename(ref reference.Reference, filename string) (types.ImageManifest, error) {
	bytes, err := os.ReadFile(filename)
	switch {
	case os.IsNotExist(err):
		return types.ImageManifest{}, errors.Wrapf(types.ErrManifestNotFound, "%q does not exist", ref.String())
	case err != nil:
		return types.ImageManifest{}, err
	}
	var manifestInfo struct {
		types.ImageManifest

		// Deprecated Fields, replaced by Descriptor
		Digest   digest.Digest
		Platform *manifestlist.PlatformSpec
	}

	if err := json.Unmarshal(bytes, &manifestInfo); err != nil {
		return types.ImageManifest{}, err
	}

	// Compatibility with image manifests created before
	// descriptor, newer versions omit Digest and Platform
	if manifestInfo.Digest != "" {
		mediaType, raw, err := manifestInfo.Payload()
		if err != nil {
			return types.ImageManifest{}, err
		}
		if dgst := digest.FromBytes(raw); dgst != manifestInfo.Digest {
			return types.ImageManifest{}, errors.Errorf("invalid manifest file %v: image manifest digest mismatch (%v != %v)", filename, manifestInfo.Digest, dgst)
		}
		manifestInfo.ImageManifest.Descriptor = ocispec.Descriptor{
			Digest:    manifestInfo.Digest,
			Size:      int64(len(raw)),
			MediaType: mediaType,
			Platform:  types.OCIPlatform(manifestInfo.Platform),
		}
	}

	return manifestInfo.ImageManifest, nil
}

// GetList returns all the local manifests for a transaction
func (s *fsStore) GetList(listRef reference.Reference) ([]types.ImageManifest, error) {
	filenames, err := s.listManifests(listRef.String())
	switch {
	case err != nil:
		return nil, err
	case filenames == nil:
		return nil, errors.Wrapf(types.ErrManifestNotFound, "%q does not exist", listRef.String())
	}

	manifests := []types.ImageManifest{}
	for _, filename := range filenames {
		filename = filepath.Join(s.root, makeFilesafeName(listRef.String()), filename)
		manifest, err := s.getFromFilename(listRef, filename)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

// listManifests stored in a transaction
func (s *fsStore) listManifests(transaction string) ([]string, error) {
	transactionDir := filepath.Join(s.root, makeFilesafeName(transaction))
	fileInfos, err := os.ReadDir(transactionDir)
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	}

	filenames := make([]string, 0, len(fileInfos))
	for _, info := range fileInfos {
		filenames = append(filenames, info.Name())
	}
	return filenames, nil
}

// Save a manifest as part of a local manifest list
func (s *fsStore) Save(listRef reference.Reference, manifest reference.Reference, images ...types.ImageManifest) error {
	if err := s.createManifestListDirectory(listRef.String()); err != nil {
		return err
	}
	if len(images) == 0 {
		return nil
	}

	filename := manifestToFilename(s.root, listRef.String(), manifest.String())
	bytes, err := json.Marshal(images[0])
	if err != nil {
		return err
	}
	if err := os.WriteFile(filename, bytes, 0o644); err != nil {
		return err
	}

	for i, image := range images[1:] {
		bytes, err := json.Marshal(image)
		if err != nil {
			return err
		}
		if err := os.WriteFile(fmt.Sprintf("%s_%d", filename, i+2), bytes, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (s *fsStore) createManifestListDirectory(transaction string) error {
	path := filepath.Join(s.root, makeFilesafeName(transaction))
	return os.MkdirAll(path, 0o755)
}

func manifestToFilename(root, manifestList, manifest string) string {
	return filepath.Join(root, makeFilesafeName(manifestList), makeFilesafeName(manifest))
}

func makeFilesafeName(ref string) string {
	fileName := strings.Replace(ref, ":", "-", -1)
	return strings.Replace(fileName, "/", "_", -1)
}
