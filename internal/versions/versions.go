package versions

import (
	"context"
	"path"
	"sort"
	"strings"

	registryclient "github.com/docker/cli/cli/registry/client"
	clitypes "github.com/docker/cli/types"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	ver "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GetEngineVersions reports the versions of the engine that are available
func GetEngineVersions(ctx context.Context, registryClient registryclient.RegistryClient, registryPrefix string, serverVersion types.Version) (clitypes.AvailableVersions, error) {
	imageName := getEngineImage(registryPrefix, serverVersion)
	imageRef, err := reference.ParseNormalizedNamed(imageName)
	if err != nil {
		return clitypes.AvailableVersions{}, err
	}

	tags, err := registryClient.GetTags(ctx, imageRef)
	if err != nil {
		return clitypes.AvailableVersions{}, err
	}

	return parseTags(tags, serverVersion.Version)
}

func getEngineImage(registryPrefix string, serverVersion types.Version) string {
	platform := strings.ToLower(serverVersion.Platform.Name)
	if platform != "" {
		if strings.Contains(platform, "enterprise") {
			return path.Join(registryPrefix, clitypes.EnterpriseEngineImage)
		}
		return path.Join(registryPrefix, clitypes.CommunityEngineImage)
	}

	// TODO This check is only applicable for early 18.09 builds that had some packaging bugs
	// and can be removed once we're no longer testing with them
	if strings.Contains(serverVersion.Version, "ee") {
		return path.Join(registryPrefix, clitypes.EnterpriseEngineImage)
	}

	return path.Join(registryPrefix, clitypes.CommunityEngineImage)
}

func parseTags(tags []string, currentVersion string) (clitypes.AvailableVersions, error) {
	var ret clitypes.AvailableVersions
	currentVer, err := ver.NewVersion(currentVersion)
	if err != nil {
		return ret, errors.Wrapf(err, "failed to parse existing version %s", currentVersion)
	}
	downgrades := []clitypes.DockerVersion{}
	patches := []clitypes.DockerVersion{}
	upgrades := []clitypes.DockerVersion{}
	currentSegments := currentVer.Segments()
	for _, tag := range tags {
		tmp, err := ver.NewVersion(tag)
		if err != nil {
			logrus.Debugf("Unable to parse %s: %s", tag, err)
			continue
		}
		testVersion := clitypes.DockerVersion{Version: *tmp, Tag: tag}
		if testVersion.LessThan(currentVer) {
			downgrades = append(downgrades, testVersion)
			continue
		}
		testSegments := testVersion.Segments()
		// lib always provides min 3 segments
		if testSegments[0] == currentSegments[0] &&
			testSegments[1] == currentSegments[1] {
			patches = append(patches, testVersion)
		} else {
			upgrades = append(upgrades, testVersion)
		}
	}
	sort.Slice(downgrades, func(i, j int) bool {
		return downgrades[i].Version.LessThan(&downgrades[j].Version)
	})
	sort.Slice(patches, func(i, j int) bool {
		return patches[i].Version.LessThan(&patches[j].Version)
	})
	sort.Slice(upgrades, func(i, j int) bool {
		return upgrades[i].Version.LessThan(&upgrades[j].Version)
	})
	ret.Downgrades = downgrades
	ret.Patches = patches
	ret.Upgrades = upgrades
	return ret, nil
}
