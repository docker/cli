//go:build linux

package standalone

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/moby/moby/api/types/registry"
	"github.com/opencontainers/go-digest"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestNormalizeImageRef(t *testing.T) {
	tests := []struct {
		ref      string
		expected string
		wantErr  bool
	}{
		{ref: "alpine", expected: "docker.io/library/alpine:latest"},
		{ref: "alpine:3.22", expected: "docker.io/library/alpine:3.22"},
		{ref: "library/alpine", expected: "docker.io/library/alpine:latest"},
		{ref: "example.com:5000/foo/bar:v1", expected: "example.com:5000/foo/bar:v1"},
		{
			ref:      "alpine@sha256:a0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde",
			expected: "docker.io/library/alpine@sha256:a0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde",
		},
		{ref: "INVALID UPPER", wantErr: true},
		{ref: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			named, err := normalizeImageRef(tc.ref)
			if tc.wantErr {
				assert.Check(t, err != nil)
				return
			}
			assert.NilError(t, err)
			assert.Check(t, is.Equal(named.String(), tc.expected))
		})
	}
}

func TestFamiliarizeImageRef(t *testing.T) {
	tests := []struct{ name, expected string }{
		{name: "docker.io/library/alpine:latest", expected: "alpine:latest"},
		{name: "docker.io/user/img:v1", expected: "user/img:v1"},
		{name: "example.com:5000/foo:v1", expected: "example.com:5000/foo:v1"},
		{name: "not a reference", expected: "not a reference"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Check(t, is.Equal(familiarizeImageRef(tc.name), tc.expected))
		})
	}
}

func TestIsImageID(t *testing.T) {
	tests := []struct {
		ref      string
		expected bool
	}{
		{ref: "33bee74c45f3", expected: true},
		{ref: "sha256:33bee74c45f3", expected: true},
		{ref: "33be", expected: true},
		{ref: "alpine", expected: false},
		{ref: "abc", expected: false}, // too short
		{ref: "33bee74c45g3", expected: false},
		{ref: "", expected: false},
	}
	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			assert.Check(t, is.Equal(isImageID(tc.ref), tc.expected))
		})
	}
}

func TestMatchesImageID(t *testing.T) {
	dgst := digest.Digest("sha256:33bee74c45f307e3268adc2010c0f55c48e7a6041e12cd12432bb1a46e498e43")
	assert.Check(t, matchesImageID(dgst, "33bee74c45f3"))
	assert.Check(t, matchesImageID(dgst, "sha256:33bee74c45f3"))
	assert.Check(t, matchesImageID(dgst, dgst.Encoded()))
	assert.Check(t, !matchesImageID(dgst, "44bee74c45f3"))
}

func TestDecodeAuthConfig(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ac, err := decodeAuthConfig("")
		assert.NilError(t, err)
		assert.Check(t, is.Equal(ac, registry.AuthConfig{}))
	})

	t.Run("valid", func(t *testing.T) {
		expected := registry.AuthConfig{Username: "user", Password: "pass", ServerAddress: "example.com"}
		data, err := json.Marshal(expected)
		assert.NilError(t, err)
		ac, err := decodeAuthConfig(base64.URLEncoding.EncodeToString(data))
		assert.NilError(t, err)
		assert.Check(t, is.Equal(ac, expected))
	})

	t.Run("raw url encoding", func(t *testing.T) {
		expected := registry.AuthConfig{IdentityToken: "token"}
		data, err := json.Marshal(expected)
		assert.NilError(t, err)
		ac, err := decodeAuthConfig(base64.RawURLEncoding.EncodeToString(data))
		assert.NilError(t, err)
		assert.Check(t, is.Equal(ac, expected))
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := decodeAuthConfig("!!!not base64!!!")
		assert.Check(t, err != nil)
	})

	t.Run("not json", func(t *testing.T) {
		_, err := decodeAuthConfig(base64.URLEncoding.EncodeToString([]byte("plain")))
		assert.Check(t, err != nil)
	})
}
