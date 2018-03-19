package containerizedengine

import (
	"context"
	"testing"

	"gotest.tools/assert"
)

func TestGetEngineVersionsBadImage(t *testing.T) {
	ctx := context.Background()
	client := baseClient{}

	currentVersion := "currentversiongoeshere"
	imageName := "this is an illegal image $%^&"
	_, err := client.GetEngineVersions(ctx, nil, currentVersion, imageName)
	assert.ErrorContains(t, err, "invalid reference format")
}

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
