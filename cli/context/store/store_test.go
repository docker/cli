// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package store

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/docker/docker/errdefs"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type endpoint struct {
	Foo string `json:"a_very_recognizable_field_name"`
}

type context struct {
	Bar string `json:"another_very_recognizable_field_name"`
}

var testCfg = NewConfig(func() any { return &context{} },
	EndpointTypeGetter("ep1", func() any { return &endpoint{} }),
	EndpointTypeGetter("ep2", func() any { return &endpoint{} }),
)

func TestExportImport(t *testing.T) {
	s := New(t.TempDir(), testCfg)
	err := s.CreateOrUpdate(
		Metadata{
			Endpoints: map[string]any{
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
	s := New(t.TempDir(), testCfg)
	err := s.CreateOrUpdate(
		Metadata{
			Endpoints: map[string]any{
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
	assert.Check(t, is.ErrorType(err, errdefs.IsNotFound))
	f, err := s.ListTLSFiles("source")
	assert.NilError(t, err)
	assert.Equal(t, 0, len(f))
}

func TestListEmptyStore(t *testing.T) {
	result, err := New(t.TempDir(), testCfg).List()
	assert.NilError(t, err)
	assert.Check(t, len(result) == 0)
}

func TestErrHasCorrectContext(t *testing.T) {
	_, err := New(t.TempDir(), testCfg).GetMetadata("no-exists")
	assert.ErrorContains(t, err, "no-exists")
	assert.Check(t, is.ErrorType(err, errdefs.IsNotFound))
}

func TestDetectImportContentType(t *testing.T) {
	buf := new(bytes.Buffer)
	r := bufio.NewReader(buf)
	ct, err := getImportContentType(r)
	assert.NilError(t, err)
	assert.Assert(t, zipType != ct)
}

func TestImportTarInvalid(t *testing.T) {
	testDir := t.TempDir()
	tf := path.Join(testDir, "test.context")

	f, err := os.Create(tf)
	assert.NilError(t, err)
	defer f.Close()

	tw := tar.NewWriter(f)
	hdr := &tar.Header{
		Name: "dummy-file",
		Mode: 0o600,
		Size: int64(len("hello world")),
	}
	err = tw.WriteHeader(hdr)
	assert.NilError(t, err)
	_, err = tw.Write([]byte("hello world"))
	assert.NilError(t, err)
	err = tw.Close()
	assert.NilError(t, err)

	source, err := os.Open(tf)
	assert.NilError(t, err)
	defer source.Close()
	var r io.Reader = source
	s := New(testDir, testCfg)
	err = Import("tarInvalid", s, r)
	assert.ErrorContains(t, err, "unexpected context file")
}

func TestImportZip(t *testing.T) {
	testDir := t.TempDir()
	zf := path.Join(testDir, "test.zip")

	f, err := os.Create(zf)
	assert.NilError(t, err)
	defer f.Close()
	w := zip.NewWriter(f)

	meta, err := json.Marshal(Metadata{
		Endpoints: map[string]any{
			"ep1": endpoint{Foo: "bar"},
		},
		Metadata: context{Bar: "baz"},
		Name:     "source",
	})
	assert.NilError(t, err)
	files := []struct {
		Name, Body string
	}{
		{"meta.json", string(meta)},
		{path.Join("tls", "docker", "ca.pem"), string([]byte("ca.pem"))},
	}

	for _, file := range files {
		f, err := w.Create(file.Name)
		assert.NilError(t, err)
		_, err = f.Write([]byte(file.Body))
		assert.NilError(t, err)
	}

	err = w.Close()
	assert.NilError(t, err)

	source, err := os.Open(zf)
	assert.NilError(t, err)
	ct, err := getImportContentType(bufio.NewReader(source))
	assert.NilError(t, err)
	assert.Equal(t, zipType, ct)

	source, _ = os.Open(zf)
	defer source.Close()
	var r io.Reader = source
	s := New(testDir, testCfg)
	err = Import("zipTest", s, r)
	assert.NilError(t, err)
}

func TestImportZipInvalid(t *testing.T) {
	testDir := t.TempDir()
	zf := path.Join(testDir, "test.zip")

	f, err := os.Create(zf)
	assert.NilError(t, err)
	defer f.Close()
	w := zip.NewWriter(f)

	df, err := w.Create("dummy-file")
	assert.NilError(t, err)
	_, err = df.Write([]byte("hello world"))
	assert.NilError(t, err)
	err = w.Close()
	assert.NilError(t, err)

	source, err := os.Open(zf)
	assert.NilError(t, err)
	defer source.Close()
	var r io.Reader = source
	s := New(testDir, testCfg)
	err = Import("zipInvalid", s, r)
	assert.ErrorContains(t, err, "unexpected context file")
}

func TestCorruptMetadata(t *testing.T) {
	tempDir := t.TempDir()
	s := New(tempDir, testCfg)
	err := s.CreateOrUpdate(
		Metadata{
			Endpoints: map[string]any{
				"ep1": endpoint{Foo: "bar"},
			},
			Metadata: context{Bar: "baz"},
			Name:     "source",
		})
	assert.NilError(t, err)

	// Simulate the meta.json file getting corrupted
	// by some external process.
	contextDir := s.meta.contextDir(contextdirOf("source"))
	contextFile := filepath.Join(contextDir, metaFile)
	err = os.WriteFile(contextFile, nil, 0o600)
	assert.NilError(t, err)

	// Assert that the error message gives the user some clue where to look.
	_, err = s.GetMetadata("source")
	assert.ErrorContains(t, err, fmt.Sprintf("parsing %s: unexpected end of JSON input", contextFile))
}

func TestNames(t *testing.T) {
	names, err := Names(nil)
	assert.Check(t, is.Error(err, "nil lister"))
	assert.Check(t, is.Len(names, 0))
}
