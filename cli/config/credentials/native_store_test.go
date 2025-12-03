package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/client"
	"github.com/docker/docker-credential-helpers/credentials"
	"github.com/moby/moby/api/types/registry"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

const (
	validServerAddress   = "https://index.docker.io/v1"
	validServerAddress2  = "https://example.com:5002"
	invalidServerAddress = "https://foobar.example.com"
	missingCredsAddress  = "https://missing.docker.io/v1"
)

var errCommandExited = errors.New("exited 1")

// mockCommand simulates interactions between the docker client and a remote
// credentials helper.
// Unit tests inject this mocked command into the remote to control execution.
type mockCommand struct {
	arg   string
	input io.Reader
}

// Output returns responses from the remote credentials helper.
// It mocks those responses based in the input in the mock.
func (m *mockCommand) Output() ([]byte, error) {
	in, err := io.ReadAll(m.input)
	if err != nil {
		return nil, err
	}
	inS := string(in)

	switch m.arg {
	case "erase":
		switch inS {
		case validServerAddress:
			return nil, nil
		default:
			return []byte("program failed"), errCommandExited
		}
	case "get":
		switch inS {
		case validServerAddress:
			return []byte(`{"Username": "foo", "Secret": "bar"}`), nil
		case validServerAddress2:
			return []byte(`{"Username": "<token>", "Secret": "abcd1234"}`), nil
		case missingCredsAddress:
			return []byte(credentials.NewErrCredentialsNotFound().Error()), errCommandExited
		case invalidServerAddress:
			return []byte("program failed"), errCommandExited
		}
	case "store":
		var c credentials.Credentials
		err := json.NewDecoder(strings.NewReader(inS)).Decode(&c)
		if err != nil {
			return []byte("program failed"), errCommandExited
		}
		switch c.ServerURL {
		case validServerAddress:
			return nil, nil
		default:
			return []byte("program failed"), errCommandExited
		}
	case "list":
		return []byte(fmt.Sprintf(`{"%s": "%s", "%s": "%s"}`, validServerAddress, "foo", validServerAddress2, "<token>")), nil
	}

	return []byte(fmt.Sprintf("unknown argument %q with %q", m.arg, inS)), errCommandExited
}

// Input sets the input to send to a remote credentials helper.
func (m *mockCommand) Input(in io.Reader) {
	m.input = in
}

func mockCommandFn(args ...string) client.Program {
	return &mockCommand{
		arg: args[0],
	}
}

func TestNativeStoreAddCredentials(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{}}
	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	auth := registry.AuthConfig{
		Username:      "foo",
		Password:      "bar",
		ServerAddress: validServerAddress,
	}
	err := s.Store(auth)
	assert.NilError(t, err)
	assert.Check(t, is.Len(f.GetAuthConfigs(), 1))

	actual, ok := f.GetAuthConfigs()[validServerAddress]
	assert.Check(t, ok)
	expected := registry.AuthConfig{
		ServerAddress: auth.ServerAddress,
	}
	assert.Check(t, is.DeepEqual(expected, actual))
}

func TestNativeStoreAddInvalidCredentials(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{}}
	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	err := s.Store(registry.AuthConfig{
		Username:      "foo",
		Password:      "bar",
		ServerAddress: invalidServerAddress,
	})
	assert.ErrorContains(t, err, "program failed")
	assert.Check(t, is.Len(f.GetAuthConfigs(), 0))
}

func TestNativeStoreGet(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{
		validServerAddress: {
			Username: "foo@example.com",
		},
	}}
	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	actual, err := s.Get(validServerAddress)
	assert.NilError(t, err)

	expected := registry.AuthConfig{
		Username:      "foo",
		Password:      "bar",
		ServerAddress: validServerAddress,
	}
	assert.Check(t, is.DeepEqual(expected, actual))
}

func TestNativeStoreGetIdentityToken(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{
		validServerAddress2: {},
	}}

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	actual, err := s.Get(validServerAddress2)
	assert.NilError(t, err)

	expected := registry.AuthConfig{
		IdentityToken: "abcd1234",
		ServerAddress: validServerAddress2,
	}
	assert.Check(t, is.DeepEqual(expected, actual))
}

func TestNativeStoreGetAll(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{
		validServerAddress: {},
	}}

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	as, err := s.GetAll()
	assert.NilError(t, err)
	assert.Check(t, is.Len(as, 2))
	expected := registry.AuthConfig{
		Username:      "foo",
		Password:      "bar",
		ServerAddress: "https://index.docker.io/v1",
		IdentityToken: "",
	}
	actual, ok := as[validServerAddress]
	assert.Check(t, ok)
	assert.Check(t, is.DeepEqual(expected, actual))
}

func TestNativeStoreGetMissingCredentials(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{
		validServerAddress: {},
	}}

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	_, err := s.Get(missingCredsAddress)
	assert.NilError(t, err)
}

func TestNativeStoreGetInvalidAddress(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{
		validServerAddress: {},
	}}

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	_, err := s.Get(invalidServerAddress)
	assert.ErrorContains(t, err, "program failed")
}

func TestNativeStoreErase(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{
		validServerAddress: {},
	}}

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	err := s.Erase(validServerAddress)
	assert.NilError(t, err)
	assert.Check(t, is.Len(f.GetAuthConfigs(), 0))
}

func TestNativeStoreEraseInvalidAddress(t *testing.T) {
	f := &fakeStore{configs: map[string]registry.AuthConfig{
		validServerAddress: {},
	}}

	s := &nativeStore{
		programFunc: mockCommandFn,
		fileStore:   NewFileStore(f),
	}
	err := s.Erase(invalidServerAddress)
	assert.ErrorContains(t, err, "program failed")
}
