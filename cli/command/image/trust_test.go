package image

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/cli/cli/trust"
	"github.com/docker/cli/internal/test"
	"github.com/docker/cli/internal/test/testutil"
	"github.com/docker/distribution/reference"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
	"golang.org/x/net/context"
)

func unsetENV() {
	os.Unsetenv("DOCKER_CONTENT_TRUST")
	os.Unsetenv("DOCKER_CONTENT_TRUST_SERVER")
}

func TestTrustServerFromEnv(t *testing.T) {
	defer unsetENV()
	indexInfo := &registrytypes.IndexInfo{Name: "testserver"}
	require.NoError(t, os.Setenv("DOCKER_CONTENT_TRUST_SERVER", "https://notary-test.com:5000"))

	expectedStr := "https://notary-test.com:5000"
	output, err := trust.Server(indexInfo)
	require.NoError(t, err)
	assert.Equal(t, expectedStr, output)
}

func TestTrustServerNotHTTPS(t *testing.T) {
	defer unsetENV()
	indexInfo := &registrytypes.IndexInfo{Name: "testserver"}
	require.NoError(t, os.Setenv("DOCKER_CONTENT_TRUST_SERVER", "http://notary-test.com:5000"))

	_, err := trust.Server(indexInfo)
	testutil.ErrorContains(t, err, "valid https URL required for trust server")
}

func TestTrustServerOfficial(t *testing.T) {
	indexInfo := &registrytypes.IndexInfo{Name: "testserver", Official: true}

	output, err := trust.Server(indexInfo)
	require.NoError(t, err)
	assert.Equal(t, registry.NotaryServer, output)
}

func TestTrustServerNonOfficial(t *testing.T) {
	indexInfo := &registrytypes.IndexInfo{Name: "testserver", Official: false}
	expectedStr := "https://" + indexInfo.Name

	output, err := trust.Server(indexInfo)
	require.NoError(t, err)
	assert.Equal(t, expectedStr, output)
}

func TestTagTrusted(t *testing.T) {
	ctx := context.Background()
	expectedFrom := "alpine@sha256:f02537b30c729bb1ee0c67c8bff3a28972bed0abbee8f5520421994c5247b896"
	expectedTo := "example/app:v1"
	fakeClient := &fakeClient{
		imageTagFunc: func(from string, to string) error {
			assert.Equal(t, expectedFrom, from)
			assert.Equal(t, expectedTo, to)
			return nil
		},
	}
	cli := test.NewFakeCli(fakeClient)

	canonical, err := reference.Parse("docker.io/library/alpine@sha256:f02537b30c729bb1ee0c67c8bff3a28972bed0abbee8f5520421994c5247b896")
	require.NoError(t, err)
	namedTag, err := reference.Parse("docker.io/example/app:v1")
	require.NoError(t, err)

	err = TagTrusted(ctx, cli, canonical, namedTag)
	require.NoError(t, err)
	expectedOut := fmt.Sprintf("Tagging %s as %s\n", expectedFrom, expectedTo)
	assert.Equal(t, expectedOut, cli.ErrBuffer().String())
}

func TestAddTargetToAllSignableRolesError(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "notary-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	notaryRepo, err := client.NewFileCachedRepository(tmpDir, "gun", "https://localhost", nil, passphrase.ConstantRetriever("password"), trustpinning.TrustPinConfig{})
	require.NoError(t, err)
	target := client.Target{}
	err = AddTargetToAllSignableRoles(notaryRepo, &target)
	assert.EqualError(t, err, "client is offline")
}
