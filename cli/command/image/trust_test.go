package image

import (
	"testing"

	"github.com/docker/cli/cli/trust"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/passphrase"
	"github.com/theupdateframework/notary/trustpinning"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
)

func TestENVTrustServer(t *testing.T) {
	defer env.PatchAll(t, map[string]string{"DOCKER_CONTENT_TRUST_SERVER": "https://notary-test.example.com:5000"})()
	indexInfo := &registrytypes.IndexInfo{Name: "testserver"}
	output, err := trust.Server(indexInfo)
	expectedStr := "https://notary-test.example.com:5000"
	if err != nil || output != expectedStr {
		t.Fatalf("Expected server to be %s, got %s", expectedStr, output)
	}
}

func TestHTTPENVTrustServer(t *testing.T) {
	defer env.PatchAll(t, map[string]string{"DOCKER_CONTENT_TRUST_SERVER": "http://notary-test.example.com:5000"})()
	indexInfo := &registrytypes.IndexInfo{Name: "testserver"}
	_, err := trust.Server(indexInfo)
	if err == nil {
		t.Fatal("Expected error with invalid scheme")
	}
}

func TestOfficialTrustServer(t *testing.T) {
	indexInfo := &registrytypes.IndexInfo{Name: "testserver", Official: true}
	output, err := trust.Server(indexInfo)
	if err != nil || output != trust.NotaryServer {
		t.Fatalf("Expected server to be %s, got %s", trust.NotaryServer, output)
	}
}

func TestNonOfficialTrustServer(t *testing.T) {
	indexInfo := &registrytypes.IndexInfo{Name: "testserver", Official: false}
	output, err := trust.Server(indexInfo)
	expectedStr := "https://" + indexInfo.Name
	if err != nil || output != expectedStr {
		t.Fatalf("Expected server to be %s, got %s", expectedStr, output)
	}
}

func TestAddTargetToAllSignableRolesError(t *testing.T) {
	notaryRepo, err := client.NewFileCachedRepository(t.TempDir(), "gun", "https://localhost", nil, passphrase.ConstantRetriever("password"), trustpinning.TrustPinConfig{})
	assert.NilError(t, err)
	target := client.Target{}
	err = AddTargetToAllSignableRoles(notaryRepo, &target)
	assert.Error(t, err, "client is offline")
}
