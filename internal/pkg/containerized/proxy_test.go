package containerized

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
)

func TestUpdateConfigNotExist(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "cfg-update")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	origProxyDir := proxydir
	defer func() {
		proxydir = origProxyDir
	}()
	proxydir = tmpdir
	name := "myname"
	newImage := "newimage:foo"
	err = updateConfig(name, newImage)
	assert.NilError(t, err)
}

func TestUpdateConfigBadJson(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "cfg-update")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	origProxyDir := proxydir
	defer func() {
		proxydir = origProxyDir
	}()
	proxydir = tmpdir
	filename := filepath.Join(tmpdir, "dockerd.json")
	err = ioutil.WriteFile(filename, []byte("not json"), 0644)
	assert.NilError(t, err)
	name := "dockerd"
	newImage := "newimage:foo"
	err = updateConfig(name, newImage)
	assert.ErrorContains(t, err, "invalid character")
}

func TestUpdateConfigHappyPath(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "cfg-update")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	origProxyDir := proxydir
	defer func() {
		proxydir = origProxyDir
	}()
	proxydir = tmpdir
	filename := filepath.Join(tmpdir, "dockerd.json")
	err = ioutil.WriteFile(filename, []byte("{}"), 0644)
	assert.NilError(t, err)
	name := "dockerd"
	newImage := "newimage:foo"
	err = updateConfig(name, newImage)
	assert.NilError(t, err)
	data, err := ioutil.ReadFile(filename)
	assert.NilError(t, err)
	var cfg map[string]string
	err = json.Unmarshal(data, &cfg)
	assert.NilError(t, err)
	assert.Assert(t, cfg["image"] == newImage)
}
