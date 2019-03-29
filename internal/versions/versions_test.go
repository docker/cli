package versions

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	clitypes "github.com/docker/cli/types"
	"gotest.tools/assert"
)

func TestParseTagsSimple(t *testing.T) {
	tags := []string{"1.0.0", "1.1.2", "1.1.1", "1.2.2"}
	currentVersion := "1.1.0"
	res, err := parseTags(tags, currentVersion)
	assert.NilError(t, err)

	assert.Assert(t, err, "already present")
	assert.Assert(t, len(res.Downgrades) == 1 && res.Downgrades[0].Tag == "1.0.0")
	assert.Assert(t, len(res.Patches) == 2 && res.Patches[0].Tag == "1.1.1" && res.Patches[1].Tag == "1.1.2")
	assert.Assert(t, len(res.Upgrades) == 1 && res.Upgrades[0].Tag == "1.2.2")
}

func TestParseConfirmMinSegments(t *testing.T) {
	tags := []string{"1", "1.1.1", "2"}
	currentVersion := "1.1"
	res, err := parseTags(tags, currentVersion)
	assert.NilError(t, err)

	assert.Assert(t, err, "already present")
	assert.Assert(t, len(res.Downgrades) == 1 && res.Downgrades[0].Tag == "1")
	assert.Assert(t, len(res.Patches) == 1 && res.Patches[0].Tag == "1.1.1")
	assert.Assert(t, len(res.Upgrades) == 1 && res.Upgrades[0].Tag == "2")
}

func TestParseTagsFilterPrerelease(t *testing.T) {
	tags := []string{"1.0.0", "1.1.1", "1.2.2", "1.1.0-beta1"}
	currentVersion := "1.1.0"
	res, err := parseTags(tags, currentVersion)
	assert.NilError(t, err)

	assert.Assert(t, err, "already present")
	assert.Assert(t, len(res.Downgrades) == 2 && res.Downgrades[0].Tag == "1.0.0")
	assert.Assert(t, len(res.Patches) == 1 && res.Patches[0].Tag == "1.1.1")
	assert.Assert(t, len(res.Upgrades) == 1 && res.Upgrades[0].Tag == "1.2.2")
}

func TestParseTagsBadTag(t *testing.T) {
	tags := []string{"1.0.0", "1.1.1", "1.2.2", "notasemanticversion"}
	currentVersion := "1.1.0"
	res, err := parseTags(tags, currentVersion)
	assert.NilError(t, err)

	assert.Assert(t, err, "already present")
	assert.Assert(t, len(res.Downgrades) == 1 && res.Downgrades[0].Tag == "1.0.0")
	assert.Assert(t, len(res.Patches) == 1 && res.Patches[0].Tag == "1.1.1")
	assert.Assert(t, len(res.Upgrades) == 1 && res.Upgrades[0].Tag == "1.2.2")
}

func TestParseBadCurrent(t *testing.T) {
	tags := []string{"1.0.0", "1.1.2", "1.1.1", "1.2.2"}
	currentVersion := "notasemanticversion"
	_, err := parseTags(tags, currentVersion)
	assert.ErrorContains(t, err, "failed to parse existing")
}

func TestParseBadCurrent2(t *testing.T) {
	tags := []string{"1.0.0", "1.1.2", "1.1.1", "1.2.2"}
	currentVersion := ""
	_, err := parseTags(tags, currentVersion)
	assert.ErrorContains(t, err, "failed to parse existing")
}

func TestGetCurrentRuntimeMetadataNotPresent(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "docker-root")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	_, err = GetCurrentRuntimeMetadata(tmpdir)
	assert.ErrorType(t, err, os.IsNotExist)
}

func TestGetCurrentRuntimeMetadataBadJson(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "docker-root")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	filename := filepath.Join(tmpdir, clitypes.RuntimeMetadataName+".json")
	err = ioutil.WriteFile(filename, []byte("not json"), 0644)
	assert.NilError(t, err)
	_, err = GetCurrentRuntimeMetadata(tmpdir)
	assert.ErrorContains(t, err, "malformed runtime metadata file")
}

func TestGetCurrentRuntimeMetadataHappyPath(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "docker-root")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)
	metadata := clitypes.RuntimeMetadata{Platform: "platformgoeshere"}
	err = WriteRuntimeMetadata(tmpdir, &metadata)
	assert.NilError(t, err)

	res, err := GetCurrentRuntimeMetadata(tmpdir)
	assert.NilError(t, err)
	assert.Equal(t, res.Platform, "platformgoeshere")
}
