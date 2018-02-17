package types

import (
	"encoding/json"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

// ImageManifest contains info to output for a manifest object.
type ImageManifest struct {
	Ref              *SerializableNamed
	Digest           digest.Digest
	SchemaV2Manifest *schema2.DeserializedManifest `json:",omitempty"`
	Platform         manifestlist.PlatformSpec
}

// Blobs returns the digests for all the blobs referenced by this manifest
func (i ImageManifest) Blobs() []digest.Digest {
	digests := []digest.Digest{}
	for _, descriptor := range i.SchemaV2Manifest.References() {
		digests = append(digests, descriptor.Digest)
	}
	return digests
}

// Payload returns the media type and bytes for the manifest
func (i ImageManifest) Payload() (string, []byte, error) {
	switch {
	case i.SchemaV2Manifest != nil:
		return i.SchemaV2Manifest.Payload()
	default:
		return "", nil, errors.Errorf("%s has no payload", i.Ref)
	}
}

// References implements the distribution.Manifest interface. It delegates to
// the underlying manifest.
func (i ImageManifest) References() []distribution.Descriptor {
	switch {
	case i.SchemaV2Manifest != nil:
		return i.SchemaV2Manifest.References()
	default:
		return nil
	}
}

// NewImageManifest returns a new ImageManifest object. The values for Platform
// are initialized from those in the image
func NewImageManifest(ref reference.Named, digest digest.Digest, img Image, manifest *schema2.DeserializedManifest) ImageManifest {
	platform := manifestlist.PlatformSpec{
		OS:           img.OS,
		Architecture: img.Architecture,
		OSVersion:    img.OSVersion,
		OSFeatures:   img.OSFeatures,
	}
	return ImageManifest{
		Ref:              &SerializableNamed{Named: ref},
		Digest:           digest,
		SchemaV2Manifest: manifest,
		Platform:         platform,
	}
}

// SerializableNamed is a reference.Named that can be serialzied and deserialized
// from JSON
type SerializableNamed struct {
	reference.Named
}

// UnmarshalJSON loads the Named reference from JSON bytes
func (s *SerializableNamed) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return errors.Wrapf(err, "invalid named reference bytes: %s", b)
	}
	var err error
	s.Named, err = reference.ParseNamed(raw)
	return err
}

// MarshalJSON returns the JSON bytes representation
func (s *SerializableNamed) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Image is the minimal set of fields required to set default platform settings
// on a manifest.
type Image struct {
	Architecture string   `json:"architecture,omitempty"`
	OS           string   `json:"os,omitempty"`
	OSVersion    string   `json:"os.version,omitempty"`
	OSFeatures   []string `json:"os.features,omitempty"`
}

// NewImageFromJSON creates an Image configuration from json.
func NewImageFromJSON(src []byte) (*Image, error) {
	img := &Image{}
	if err := json.Unmarshal(src, img); err != nil {
		return nil, err
	}
	return img, nil
}
