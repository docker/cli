package manifest

import (
	"context"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/manifest/types"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

type osArch struct {
	os   string
	arch string
}

// Remove any unsupported os/arch combo
// list of valid os/arch values (see "Optional Environment Variables" section
// of https://golang.org/doc/install/source
// Added linux/s390x as we know System z support already exists
// Keep in sync with _docker_manifest_annotate in contrib/completion/bash/docker
var validOSArches = map[osArch]bool{
	{os: "darwin", arch: "386"}:      true,
	{os: "darwin", arch: "amd64"}:    true,
	{os: "darwin", arch: "arm"}:      true,
	{os: "darwin", arch: "arm64"}:    true,
	{os: "dragonfly", arch: "amd64"}: true,
	{os: "freebsd", arch: "386"}:     true,
	{os: "freebsd", arch: "amd64"}:   true,
	{os: "freebsd", arch: "arm"}:     true,
	{os: "linux", arch: "386"}:       true,
	{os: "linux", arch: "amd64"}:     true,
	{os: "linux", arch: "arm"}:       true,
	{os: "linux", arch: "arm64"}:     true,
	{os: "linux", arch: "ppc64le"}:   true,
	{os: "linux", arch: "mips64"}:    true,
	{os: "linux", arch: "mips64le"}:  true,
	{os: "linux", arch: "riscv64"}:   true,
	{os: "linux", arch: "s390x"}:     true,
	{os: "netbsd", arch: "386"}:      true,
	{os: "netbsd", arch: "amd64"}:    true,
	{os: "netbsd", arch: "arm"}:      true,
	{os: "openbsd", arch: "386"}:     true,
	{os: "openbsd", arch: "amd64"}:   true,
	{os: "openbsd", arch: "arm"}:     true,
	{os: "plan9", arch: "386"}:       true,
	{os: "plan9", arch: "amd64"}:     true,
	{os: "solaris", arch: "amd64"}:   true,
	{os: "windows", arch: "386"}:     true,
	{os: "windows", arch: "amd64"}:   true,
}

func isValidOSArch(os string, arch string) bool {
	// check for existence of this combo
	_, ok := validOSArches[osArch{os, arch}]
	return ok
}

func normalizeReference(ref string) (reference.Named, error) {
	namedRef, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return nil, err
	}
	if _, isDigested := namedRef.(reference.Canonical); !isDigested {
		return reference.TagNameOnly(namedRef), nil
	}
	return namedRef, nil
}

// getManifests from the local store, and fallback to the remote registry if it
// doesn't exist locally
func getManifests(ctx context.Context, dockerCli command.Cli, listRef, namedRef reference.Named, insecure bool) ([]types.ImageManifest, error) {
	// load from the local store
	if listRef != nil {
		data, err := dockerCli.ManifestStore().Get(listRef, namedRef)
		if err == nil {
			return data, nil
		} else if !errors.Is(err, types.ErrManifestNotFound) {
			return nil, err
		}
	}
	datas, err := dockerCli.ManifestStore().GetList(namedRef)
	if err == nil {
		return datas, nil
	} else if !errors.Is(err, types.ErrManifestNotFound) {
		return nil, err
	}

	// load from the remote registry
	client := dockerCli.RegistryClient(insecure)
	if client != nil {
		data, err := client.GetManifest(ctx, namedRef)
		if err == nil {
			return []types.ImageManifest{data}, nil
		} else if !errors.Is(err, types.ErrManifestNotFound) {
			return nil, err
		}

		datas, err = client.GetManifestList(ctx, namedRef)
		if err == nil {
			return datas, nil
		} else if !errors.Is(err, types.ErrManifestNotFound) {
			return nil, err
		}
	}

	return nil, errors.Wrapf(types.ErrManifestNotFound, "%q does not exist", namedRef)
}
