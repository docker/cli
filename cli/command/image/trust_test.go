package image

import (
	"testing"

	"github.com/docker/cli/cli/trust"
	registrytypes "github.com/docker/docker/api/types/registry"
	"gotest.tools/v3/env"
)

func TestENVTrustServer(t *testing.T) {
	env.PatchAll(t, map[string]string{"DOCKER_CONTENT_TRUST_SERVER": "https://notary-test.example.com:5000"})
	indexInfo := &registrytypes.IndexInfo{Name: "testserver"}
	output, err := trust.Server(indexInfo)
	expectedStr := "https://notary-test.example.com:5000"
	if err != nil || output != expectedStr {
		t.Fatalf("Expected server to be %s, got %s", expectedStr, output)
	}
}

func TestHTTPENVTrustServer(t *testing.T) {
	env.PatchAll(t, map[string]string{"DOCKER_CONTENT_TRUST_SERVER": "http://notary-test.example.com:5000"})
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
