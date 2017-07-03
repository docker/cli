package swarm

import (
	"bytes"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func swarmSpecWithFullCAConfig() *swarm.Spec {
	return &swarm.Spec{
		CAConfig: swarm.CAConfig{
			SigningCACert:  "cacert",
			SigningCAKey:   "cakey",
			ForceRotate:    1,
			NodeCertExpiry: time.Duration(200),
			ExternalCAs: []*swarm.ExternalCA{
				{
					URL:      "https://example.com/ca",
					Protocol: swarm.ExternalCAProtocolCFSSL,
					CACert:   "excacert",
				},
			},
		},
	}
}

func TestDisplayTrustRootNoRoot(t *testing.T) {
	buffer := new(bytes.Buffer)
	err := displayTrustRoot(buffer, swarm.Swarm{})
	assert.EqualError(t, err, "No CA information available")
}

func TestDisplayTrustRoot(t *testing.T) {
	buffer := new(bytes.Buffer)
	trustRoot := "trustme"
	err := displayTrustRoot(buffer, swarm.Swarm{
		ClusterInfo: swarm.ClusterInfo{
			TLSInfo: swarm.TLSInfo{TrustRoot: trustRoot},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, trustRoot+"\n", buffer.String())
}

func TestUpdateSwarmSpecDefaultRotate(t *testing.T) {
	spec := swarmSpecWithFullCAConfig()
	flags := newCACommand(nil).Flags()
	updateSwarmSpec(spec, flags, caOptions{})

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.ForceRotate = 2
	expected.CAConfig.SigningCACert = ""
	expected.CAConfig.SigningCAKey = ""
	assert.Equal(t, expected, spec)
}

func TestUpdateSwarmSpecPartial(t *testing.T) {
	spec := swarmSpecWithFullCAConfig()
	flags := newCACommand(nil).Flags()
	updateSwarmSpec(spec, flags, caOptions{
		rootCACert: PEMFile{contents: "cacert"},
	})

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.SigningCACert = "cacert"
	assert.Equal(t, expected, spec)
}

func TestUpdateSwarmSpecFullFlags(t *testing.T) {
	flags := newCACommand(nil).Flags()
	flags.Lookup(flagCertExpiry).Changed = true
	spec := swarmSpecWithFullCAConfig()
	updateSwarmSpec(spec, flags, caOptions{
		rootCACert:     PEMFile{contents: "cacert"},
		rootCAKey:      PEMFile{contents: "cakey"},
		swarmCAOptions: swarmCAOptions{nodeCertExpiry: 3 * time.Minute},
	})

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.SigningCACert = "cacert"
	expected.CAConfig.SigningCAKey = "cakey"
	expected.CAConfig.NodeCertExpiry = 3 * time.Minute
	assert.Equal(t, expected, spec)
}
