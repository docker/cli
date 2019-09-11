package command

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/go-connections/tlsconfig"
	"gotest.tools/assert"
	"gotest.tools/env"
	"gotest.tools/golden"
)

type endpoint struct {
	Foo string `json:"a_very_recognizable_field_name"`
}

type testContext struct {
	Bar string `json:"another_very_recognizable_field_name"`
}

var testCfg = store.NewConfig(func() interface{} { return &testContext{} },
	store.EndpointTypeGetter("ep1", func() interface{} { return &endpoint{} }),
	store.EndpointTypeGetter("ep2", func() interface{} { return &endpoint{} }),
)

func testDefaultMetadata() store.Metadata {
	return store.Metadata{
		Endpoints: map[string]interface{}{
			"ep1": endpoint{Foo: "bar"},
		},
		Metadata: testContext{Bar: "baz"},
		Name:     DefaultContextName,
	}
}

func testStore(t *testing.T, meta store.Metadata, tls store.ContextTLSData) (store.Store, func()) {
	//meta := testDefaultMetadata()
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	//defer os.RemoveAll(testDir)
	store := &ContextStoreWithDefault{
		Store: store.New(testDir, testCfg),
		Resolver: func() (*DefaultContext, error) {
			return &DefaultContext{
				Meta: meta,
				TLS:  tls,
			}, nil
		},
	}
	return store, func() {
		os.RemoveAll(testDir)
	}
}

func TestDefaultContextInitializer(t *testing.T) {
	cli, err := NewDockerCli()
	assert.NilError(t, err)
	defer env.Patch(t, "DOCKER_HOST", "ssh://someswarmserver")()
	cli.configFile = &configfile.ConfigFile{
		StackOrchestrator: "swarm",
	}
	ctx, err := ResolveDefaultContext(&cliflags.CommonOptions{
		TLS: true,
		TLSOptions: &tlsconfig.Options{
			CAFile: "./testdata/ca.pem",
		},
	}, cli.ConfigFile(), DefaultContextStoreConfig(), cli.Err())
	assert.NilError(t, err)
	assert.Equal(t, "default", ctx.Meta.Name)
	assert.Equal(t, OrchestratorSwarm, ctx.Meta.Metadata.(DockerContext).StackOrchestrator)
	assert.DeepEqual(t, "ssh://someswarmserver", ctx.Meta.Endpoints[docker.DockerEndpoint].(docker.EndpointMeta).Host)
	golden.Assert(t, string(ctx.TLS.Endpoints[docker.DockerEndpoint].Files["ca.pem"]), "ca.pem")
}

func TestExportDefaultImport(t *testing.T) {
	file1 := make([]byte, 1500)
	rand.Read(file1)
	file2 := make([]byte, 3700)
	rand.Read(file2)
	s, cleanup := testStore(t, testDefaultMetadata(), store.ContextTLSData{
		Endpoints: map[string]store.EndpointTLSData{
			"ep2": {
				Files: map[string][]byte{
					"file1": file1,
					"file2": file2,
				},
			},
		},
	})
	defer cleanup()
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
	s, cleanup := testStore(t, meta, store.ContextTLSData{})
	defer cleanup()
	result, err := s.List()
	assert.NilError(t, err)
	assert.Equal(t, 1, len(result))
	assert.DeepEqual(t, meta, result[0])
}

func TestGetDefaultContextStorageInfo(t *testing.T) {
	s, cleanup := testStore(t, testDefaultMetadata(), store.ContextTLSData{})
	defer cleanup()
	result := s.GetStorageInfo(DefaultContextName)
	assert.Equal(t, "<IN MEMORY>", result.MetadataPath)
	assert.Equal(t, "<IN MEMORY>", result.TLSPath)
}

func TestGetDefaultContextMetadata(t *testing.T) {
	meta := testDefaultMetadata()
	s, cleanup := testStore(t, meta, store.ContextTLSData{})
	defer cleanup()
	result, err := s.GetMetadata(DefaultContextName)
	assert.NilError(t, err)
	assert.Equal(t, DefaultContextName, result.Name)
	assert.DeepEqual(t, meta.Metadata, result.Metadata)
	assert.DeepEqual(t, meta.Endpoints, result.Endpoints)
}

func TestErrCreateDefault(t *testing.T) {
	meta := testDefaultMetadata()
	s, cleanup := testStore(t, meta, store.ContextTLSData{})
	defer cleanup()
	err := s.CreateOrUpdate(store.Metadata{
		Endpoints: map[string]interface{}{
			"ep1": endpoint{Foo: "bar"},
		},
		Metadata: testContext{Bar: "baz"},
		Name:     "default",
	})
	assert.Error(t, err, "default context cannot be created nor updated")
}

func TestErrRemoveDefault(t *testing.T) {
	meta := testDefaultMetadata()
	s, cleanup := testStore(t, meta, store.ContextTLSData{})
	defer cleanup()
	err := s.Remove("default")
	assert.Error(t, err, "default context cannot be removed")
}

func TestErrTLSDataError(t *testing.T) {
	meta := testDefaultMetadata()
	s, cleanup := testStore(t, meta, store.ContextTLSData{})
	defer cleanup()
	_, err := s.GetTLSData("default", "noop", "noop")
	assert.Check(t, store.IsErrTLSDataDoesNotExist(err))
}
