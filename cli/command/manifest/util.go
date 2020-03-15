package manifest

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/manifest/store"
	"github.com/docker/cli/cli/manifest/types"
	"github.com/docker/distribution/reference"
)

type osArch struct {
	os   string
	arch string
}

type invalidOSArchErr struct {
	osArch
}

func (e *invalidOSArchErr) Error() string {
	return fmt.Sprintf("manifest entry for image has unsupported os/arch combination: %s/%s", e.os, e.arch)
}

func newInvalidOSArchErr(os1 string, arch1 string) *invalidOSArchErr {
	return &invalidOSArchErr{osArch{os: os1, arch: arch1}}
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

func validateOSArch(os string, arch string) error {
	// check for existence of this combo
	_, ok := validOSArches[osArch{os, arch}]
	if !ok {
		return newInvalidOSArchErr(os, arch)
	}
	return nil
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

// getManifest from the local store, and fallback to the remote registry if it
//  doesn't exist locally
func getManifest(ctx context.Context, dockerCli command.Cli, listRef, namedRef reference.Named, insecure bool) (types.ImageManifest, error) {
	data, err := dockerCli.ManifestStore().Get(listRef, namedRef)
	switch {
	case store.IsNotFound(err):
		return dockerCli.RegistryClient(insecure).GetManifest(ctx, namedRef)
	case err != nil:
		return types.ImageManifest{}, err
	default:
		return data, nil
	}
}
