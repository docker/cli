package store

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func testMetadata(name string) Metadata {
	return Metadata{
		Endpoints: map[string]interface{}{
			"ep1": endpoint{Foo: "bar"},
		},
		Metadata: context{Bar: "baz"},
		Name:     name,
	}
}

func TestMetadataGetNotExisting(t *testing.T) {
	testee := metadataStore{root: t.TempDir(), config: testCfg}
	_, err := testee.get("noexist")
	assert.Assert(t, IsErrContextDoesNotExist(err))
}

func TestMetadataCreateGetRemove(t *testing.T) {
	testDir := t.TempDir()
	testee := metadataStore{root: testDir, config: testCfg}
	expected2 := Metadata{
		Endpoints: map[string]interface{}{
			"ep1": endpoint{Foo: "baz"},
			"ep2": endpoint{Foo: "bee"},
		},
		Metadata: context{Bar: "foo"},
		Name:     "test-context",
	}
	testMeta := testMetadata("test-context")
	err := testee.createOrUpdate(testMeta)
	assert.NilError(t, err)
	// create a new instance to check it does not depend on some sort of state
	testee = metadataStore{root: testDir, config: testCfg}
	meta, err := testee.get(contextdirOf("test-context"))
	assert.NilError(t, err)
	assert.DeepEqual(t, meta, testMeta)

	// update

	err = testee.createOrUpdate(expected2)
	assert.NilError(t, err)
	meta, err = testee.get(contextdirOf("test-context"))
	assert.NilError(t, err)
	assert.DeepEqual(t, meta, expected2)

	assert.NilError(t, testee.remove(contextdirOf("test-context")))
	assert.NilError(t, testee.remove(contextdirOf("test-context"))) // support duplicate remove
	_, err = testee.get(contextdirOf("test-context"))
	assert.Assert(t, IsErrContextDoesNotExist(err))
}

func TestMetadataRespectJsonAnnotation(t *testing.T) {
	testDir := t.TempDir()
	testee := metadataStore{root: testDir, config: testCfg}
	assert.NilError(t, testee.createOrUpdate(testMetadata("test")))
	bytes, err := os.ReadFile(filepath.Join(testDir, string(contextdirOf("test")), "meta.json"))
	assert.NilError(t, err)
	assert.Assert(t, cmp.Contains(string(bytes), "a_very_recognizable_field_name"))
	assert.Assert(t, cmp.Contains(string(bytes), "another_very_recognizable_field_name"))
}

func TestMetadataList(t *testing.T) {
	testee := metadataStore{root: t.TempDir(), config: testCfg}
	wholeData := []Metadata{
		testMetadata("context1"),
		testMetadata("context2"),
		testMetadata("context3"),
	}

	for _, s := range wholeData {
		assert.NilError(t, testee.createOrUpdate(s))
	}

	data, err := testee.list()
	assert.NilError(t, err)
	assert.DeepEqual(t, data, wholeData)
}

func TestEmptyConfig(t *testing.T) {
	testee := metadataStore{root: t.TempDir()}
	wholeData := []Metadata{
		testMetadata("context1"),
		testMetadata("context2"),
		testMetadata("context3"),
	}

	for _, s := range wholeData {
		assert.NilError(t, testee.createOrUpdate(s))
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
	testee := metadataStore{
		root:   t.TempDir(),
		config: NewConfig(func() interface{} { return &contextWithEmbedding{} }),
	}
	testCtxMeta := contextWithEmbedding{
		embeddedStruct: embeddedStruct{
			Val: "Hello",
		},
	}
	assert.NilError(t, testee.createOrUpdate(Metadata{Metadata: testCtxMeta, Name: "test"}))
	res, err := testee.get(contextdirOf("test"))
	assert.NilError(t, err)
	assert.Equal(t, testCtxMeta, res.Metadata)
}
