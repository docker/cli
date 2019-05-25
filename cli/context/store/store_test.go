package store

import (
	"crypto/rand"
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
	err = s.CreateOrUpdate(
		Metadata{
			Endpoints: map[string]interface{}{
				"ep1": endpoint{Foo: "bar"},
			},
			Metadata: context{Bar: "baz"},
			Name:     "source",
		})
	assert.NilError(t, err)
	file1 := make([]byte, 1500)
	rand.Read(file1)
	file2 := make([]byte, 3700)
	rand.Read(file2)
	err = s.ResetEndpointTLSMaterial("source", "ep1", &EndpointTLSData{
		Files: map[string][]byte{
			"file1": file1,
			"file2": file2,
		},
	})
	assert.NilError(t, err)
	r := Export("source", s)
	defer r.Close()
	err = Import("dest", s, r)
	assert.NilError(t, err)
	srcMeta, err := s.GetMetadata("source")
	assert.NilError(t, err)
	destMeta, err := s.GetMetadata("dest")
	assert.NilError(t, err)
	assert.DeepEqual(t, destMeta.Metadata, srcMeta.Metadata)
	assert.DeepEqual(t, destMeta.Endpoints, srcMeta.Endpoints)
	srcFileList, err := s.ListTLSFiles("source")
	assert.NilError(t, err)
	destFileList, err := s.ListTLSFiles("dest")
	assert.NilError(t, err)
	assert.Equal(t, 1, len(destFileList))
	assert.Equal(t, 1, len(srcFileList))
	assert.Equal(t, 2, len(destFileList["ep1"]))
	assert.Equal(t, 2, len(srcFileList["ep1"]))
	srcData1, err := s.GetTLSData("source", "ep1", "file1")
	assert.NilError(t, err)
	assert.DeepEqual(t, file1, srcData1)
	srcData2, err := s.GetTLSData("source", "ep1", "file2")
	assert.NilError(t, err)
	assert.DeepEqual(t, file2, srcData2)
	destData1, err := s.GetTLSData("dest", "ep1", "file1")
	assert.NilError(t, err)
	assert.DeepEqual(t, file1, destData1)
	destData2, err := s.GetTLSData("dest", "ep1", "file2")
	assert.NilError(t, err)
	assert.DeepEqual(t, file2, destData2)
}

func TestRemove(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	s := New(testDir, testCfg)
	err = s.CreateOrUpdate(
		Metadata{
			Endpoints: map[string]interface{}{
				"ep1": endpoint{Foo: "bar"},
			},
			Metadata: context{Bar: "baz"},
			Name:     "source",
		})
	assert.NilError(t, err)
	assert.NilError(t, s.ResetEndpointTLSMaterial("source", "ep1", &EndpointTLSData{
		Files: map[string][]byte{
			"file1": []byte("test-data"),
		},
	}))
	assert.NilError(t, s.Remove("source"))
	_, err = s.GetMetadata("source")
	assert.Check(t, IsErrContextDoesNotExist(err))
	f, err := s.ListTLSFiles("source")
	assert.NilError(t, err)
	assert.Equal(t, 0, len(f))
}

func TestListEmptyStore(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	store := New(testDir, testCfg)
	result, err := store.List()
	assert.NilError(t, err)
	assert.Check(t, len(result) == 0)
}

func TestErrHasCorrectContext(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	store := New(testDir, testCfg)
	_, err = store.GetMetadata("no-exists")
	assert.ErrorContains(t, err, "no-exists")
	assert.Check(t, IsErrContextDoesNotExist(err))
}
