package store

import (
	"io/ioutil"
	"os"
	"testing"

	"gotest.tools/assert"
)

type endpoint struct {
	Foo string `json:"a_very_recognizable_field_name"`
}

type context struct {
	Bar string `json:"another_very_recognizable_field_name"`
}

var testCfg = NewConfig(func() interface{} { return &context{} },
	EndpointTypeGetter("ep1", func() interface{} { return &endpoint{} }),
	EndpointTypeGetter("ep2", func() interface{} { return &endpoint{} }),
)

func TestExportImport(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	s := New(testDir, testCfg)
	err = s.CreateOrUpdateContext("source",
		ContextMetadata{
			Endpoints: map[string]interface{}{
				"ep1": endpoint{Foo: "bar"},
			},
			Metadata: context{Bar: "baz"},
		})
	assert.NilError(t, err)
	err = s.ResetContextEndpointTLSMaterial("source", "ep1", &EndpointTLSData{
		Files: map[string][]byte{
			"file1": []byte("test-data"),
		},
	})
	assert.NilError(t, err)
	r := Export("source", s)
	defer r.Close()
	err = Import("dest", s, r)
	assert.NilError(t, err)
	srcMeta, err := s.GetContextMetadata("source")
	assert.NilError(t, err)
	destMeta, err := s.GetContextMetadata("dest")
	assert.NilError(t, err)
	assert.DeepEqual(t, destMeta, srcMeta)
	srcFileList, err := s.ListContextTLSFiles("source")
	assert.NilError(t, err)
	destFileList, err := s.ListContextTLSFiles("dest")
	assert.NilError(t, err)
	assert.DeepEqual(t, srcFileList, destFileList)
	srcData, err := s.GetContextTLSData("source", "ep1", "file1")
	assert.NilError(t, err)
	assert.Equal(t, "test-data", string(srcData))
	destData, err := s.GetContextTLSData("dest", "ep1", "file1")
	assert.NilError(t, err)
	assert.Equal(t, "test-data", string(destData))
}

func TestRemove(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	s := New(testDir, testCfg)
	err = s.CreateOrUpdateContext("source",
		ContextMetadata{
			Endpoints: map[string]interface{}{
				"ep1": endpoint{Foo: "bar"},
			},
			Metadata: context{Bar: "baz"},
		})
	assert.NilError(t, err)
	assert.NilError(t, s.ResetContextEndpointTLSMaterial("source", "ep1", &EndpointTLSData{
		Files: map[string][]byte{
			"file1": []byte("test-data"),
		},
	}))
	assert.NilError(t, s.RemoveContext("source"))
	_, err = s.GetContextMetadata("source")
	assert.Check(t, IsErrContextDoesNotExist(err))
	f, err := s.ListContextTLSFiles("source")
	assert.NilError(t, err)
	assert.Equal(t, 0, len(f))
}

func TestListEmptyStore(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	store := New(testDir, testCfg)
	result, err := store.ListContexts()
	assert.NilError(t, err)
	assert.Check(t, result != nil)
	assert.Check(t, len(result) == 0)
}
