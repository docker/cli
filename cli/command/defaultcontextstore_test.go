// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package command

import (
	"crypto/rand"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/errdefs"
	"github.com/docker/go-connections/tlsconfig"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/golden"
)

type endpoint struct {
	Foo string `json:"a_very_recognizable_field_name"`
}

type testContext struct {
	Bar string `json:"another_very_recognizable_field_name"`
}

var testCfg = store.NewConfig(func() any { return &testContext{} },
	store.EndpointTypeGetter("ep1", func() any { return &endpoint{} }),
	store.EndpointTypeGetter("ep2", func() any { return &endpoint{} }),
)

func testDefaultMetadata() store.Metadata {
	return store.Metadata{
		Endpoints: map[string]any{
			"ep1": endpoint{Foo: "bar"},
		},
		Metadata: testContext{Bar: "baz"},
		Name:     DefaultContextName,
	}
}

func testStore(t *testing.T, meta store.Metadata, tls store.ContextTLSData) store.Store {
	t.Helper()
	return &ContextStoreWithDefault{
		Store: store.New(t.TempDir(), testCfg),
		Resolver: func() (*DefaultContext, error) {
			return &DefaultContext{
				Meta: meta,
				TLS:  tls,
			}, nil
		},
	}
}

func TestDefaultContextInitializer(t *testing.T) {
	cli, err := NewDockerCli()
	assert.NilError(t, err)
	t.Setenv("DOCKER_HOST", "ssh://someswarmserver")
	cli.configFile = &configfile.ConfigFile{}
	ctx, err := ResolveDefaultContext(&cliflags.ClientOptions{
		TLS: true,
		TLSOptions: &tlsconfig.Options{
			CAFile: "./testdata/ca.pem",
		},
	}, DefaultContextStoreConfig())
	assert.NilError(t, err)
	assert.Equal(t, "default", ctx.Meta.Name)
	assert.DeepEqual(t, "ssh://someswarmserver", ctx.Meta.Endpoints[docker.DockerEndpoint].(docker.EndpointMeta).Host)
	golden.Assert(t, string(ctx.TLS.Endpoints[docker.DockerEndpoint].Files["ca.pem"]), "ca.pem")
}

func TestExportDefaultImport(t *testing.T) {
	file1 := make([]byte, 1500)
	rand.Read(file1)
	file2 := make([]byte, 3700)
	rand.Read(file2)
	s := testStore(t, testDefaultMetadata(), store.ContextTLSData{
		Endpoints: map[string]store.EndpointTLSData{
			"ep2": {
				Files: map[string][]byte{
					"file1": file1,
					"file2": file2,
				},
			},
		},
	})
	r := store.Export("default", s)
	defer r.Close()
	err := store.Import("dest", s, r)
	assert.NilError(t, err)

	srcMeta, err := s.GetMetadata("default")
	assert.NilError(t, err)
	destMeta, err := s.GetMetadata("dest")
	assert.NilError(t, err)
	assert.DeepEqual(t, destMeta.Metadata, srcMeta.Metadata)
	assert.DeepEqual(t, destMeta.Endpoints, srcMeta.Endpoints)

	srcFileList, err := s.ListTLSFiles("default")
	assert.NilError(t, err)
	destFileList, err := s.ListTLSFiles("dest")
	assert.NilError(t, err)
	assert.Equal(t, 1, len(destFileList))
	assert.Equal(t, 1, len(srcFileList))
	assert.Equal(t, 2, len(destFileList["ep2"]))
	assert.Equal(t, 2, len(srcFileList["ep2"]))

	srcData1, err := s.GetTLSData("default", "ep2", "file1")
	assert.NilError(t, err)
	assert.DeepEqual(t, file1, srcData1)
	srcData2, err := s.GetTLSData("default", "ep2", "file2")
	assert.NilError(t, err)
	assert.DeepEqual(t, file2, srcData2)

	destData1, err := s.GetTLSData("dest", "ep2", "file1")
	assert.NilError(t, err)
	assert.DeepEqual(t, file1, destData1)
	destData2, err := s.GetTLSData("dest", "ep2", "file2")
	assert.NilError(t, err)
	assert.DeepEqual(t, file2, destData2)
}

func TestListDefaultContext(t *testing.T) {
	meta := testDefaultMetadata()
	s := testStore(t, meta, store.ContextTLSData{})
	result, err := s.List()
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result))
	assert.DeepEqual(t, meta, result[0])
}

func TestGetDefaultContextStorageInfo(t *testing.T) {
	s := testStore(t, testDefaultMetadata(), store.ContextTLSData{})
	result := s.GetStorageInfo(DefaultContextName)
	assert.Equal(t, "<IN MEMORY>", result.MetadataPath)
	assert.Equal(t, "<IN MEMORY>", result.TLSPath)
}

func TestGetDefaultContextMetadata(t *testing.T) {
	meta := testDefaultMetadata()
	s := testStore(t, meta, store.ContextTLSData{})
	result, err := s.GetMetadata(DefaultContextName)
	assert.NilError(t, err)
	assert.Equal(t, DefaultContextName, result.Name)
	assert.DeepEqual(t, meta.Metadata, result.Metadata)
	assert.DeepEqual(t, meta.Endpoints, result.Endpoints)
}

func TestErrCreateDefault(t *testing.T) {
	meta := testDefaultMetadata()
	s := testStore(t, meta, store.ContextTLSData{})
	err := s.CreateOrUpdate(store.Metadata{
		Endpoints: map[string]any{
			"ep1": endpoint{Foo: "bar"},
		},
		Metadata: testContext{Bar: "baz"},
		Name:     "default",
	})
	assert.Check(t, is.ErrorType(err, errdefs.IsInvalidParameter))
	assert.Error(t, err, "default context cannot be created nor updated")
}

func TestErrRemoveDefault(t *testing.T) {
	meta := testDefaultMetadata()
	s := testStore(t, meta, store.ContextTLSData{})
	err := s.Remove("default")
	assert.Check(t, is.ErrorType(err, errdefs.IsInvalidParameter))
	assert.Error(t, err, "default context cannot be removed")
}

func TestErrTLSDataError(t *testing.T) {
	meta := testDefaultMetadata()
	s := testStore(t, meta, store.ContextTLSData{})
	_, err := s.GetTLSData("default", "noop", "noop")
	assert.Check(t, is.ErrorType(err, errdefs.IsNotFound))
}
