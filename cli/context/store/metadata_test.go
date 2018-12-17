package store

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
)

var testMetadata = ContextMetadata{
	Endpoints: map[string]interface{}{
		"ep1": endpoint{Foo: "bar"},
	},
	Metadata: context{Bar: "baz"},
}

func TestMetadataGetNotExisting(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	testee := metadataStore{root: testDir, config: testCfg}
	_, err = testee.get("noexist")
	assert.Assert(t, IsErrContextDoesNotExist(err))
}

func TestMetadataCreateGetRemove(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	testee := metadataStore{root: testDir, config: testCfg}
	expected2 := ContextMetadata{
		Endpoints: map[string]interface{}{
			"ep1": endpoint{Foo: "baz"},
			"ep2": endpoint{Foo: "bee"},
		},
		Metadata: context{Bar: "foo"},
	}
	err = testee.createOrUpdate("test-context", testMetadata)
	assert.NilError(t, err)
	// create a new instance to check it does not depend on some sort of state
	testee = metadataStore{root: testDir, config: testCfg}
	meta, err := testee.get("test-context")
	assert.NilError(t, err)
	assert.DeepEqual(t, meta, testMetadata)

	// update

	err = testee.createOrUpdate("test-context", expected2)
	assert.NilError(t, err)
	meta, err = testee.get("test-context")
	assert.NilError(t, err)
	assert.DeepEqual(t, meta, expected2)

	assert.NilError(t, testee.remove("test-context"))
	assert.NilError(t, testee.remove("test-context")) // support duplicate remove
	_, err = testee.get("test-context")
	assert.Assert(t, IsErrContextDoesNotExist(err))
}

func TestMetadataRespectJsonAnnotation(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	testee := metadataStore{root: testDir, config: testCfg}
	assert.NilError(t, testee.createOrUpdate("test", testMetadata))
	bytes, err := ioutil.ReadFile(filepath.Join(testDir, "test", "meta.json"))
	assert.NilError(t, err)
	assert.Assert(t, cmp.Contains(string(bytes), "a_very_recognizable_field_name"))
	assert.Assert(t, cmp.Contains(string(bytes), "another_very_recognizable_field_name"))
}

func TestMetadataList(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	testee := metadataStore{root: testDir, config: testCfg}
	wholeData := map[string]ContextMetadata{
		"simple":                    testMetadata,
		"simple2":                   testMetadata,
		"nested/context":            testMetadata,
		"nestedwith-parent/context": testMetadata,
		"nestedwith-parent":         testMetadata,
	}

	for k, s := range wholeData {
		err = testee.createOrUpdate(k, s)
		assert.NilError(t, err)
	}

	data, err := testee.list()
	assert.NilError(t, err)
	assert.DeepEqual(t, data, wholeData)
}

func TestEmptyConfig(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	testee := metadataStore{root: testDir}
	wholeData := map[string]ContextMetadata{
		"simple":                    testMetadata,
		"simple2":                   testMetadata,
		"nested/context":            testMetadata,
		"nestedwith-parent/context": testMetadata,
		"nestedwith-parent":         testMetadata,
	}

	for k, s := range wholeData {
		err = testee.createOrUpdate(k, s)
		assert.NilError(t, err)
	}

	data, err := testee.list()
	assert.NilError(t, err)
	assert.Equal(t, len(data), len(wholeData))
}

type contextWithEmbedding struct {
	embeddedStruct
}
type embeddedStruct struct {
	Val string
}

func TestWithEmbedding(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	assert.NilError(t, err)
	defer os.RemoveAll(testDir)
	testee := metadataStore{root: testDir, config: NewConfig(func() interface{} { return &contextWithEmbedding{} })}
	testCtxMeta := contextWithEmbedding{
		embeddedStruct: embeddedStruct{
			Val: "Hello",
		},
	}
	assert.NilError(t, testee.createOrUpdate("test", ContextMetadata{Metadata: testCtxMeta}))
	res, err := testee.get("test")
	assert.NilError(t, err)
	assert.Equal(t, testCtxMeta, res.Metadata)
}
