package swarm

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
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
	assert.Error(t, err, "No CA information available")
}

func TestDisplayTrustRootInvalidFlags(t *testing.T) {
	// we need an actual PEMfile to test
	tmpfile, err := ioutil.TempFile("", "pemfile")
	assert.NilError(t, err)
	defer os.Remove(tmpfile.Name())
	tmpfile.Write([]byte(`
-----BEGIN CERTIFICATE-----
MIIBajCCARCgAwIBAgIUe0+jYWhxN8fFOByC7yveIYgvx1kwCgYIKoZIzj0EAwIw
EzERMA8GA1UEAxMIc3dhcm0tY2EwHhcNMTcwNjI3MTUxNDAwWhcNMzcwNjIyMTUx
NDAwWjATMREwDwYDVQQDEwhzd2FybS1jYTBZMBMGByqGSM49AgEGCCqGSM49AwEH
A0IABGgbOZLd7b4b262+6m4ignIecbAZKim6djNiIS1Kl5IHciXYn7gnSpsayjn7
GQABpgkdPeM9TEQowmtR1qSnORujQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMB
Af8EBTADAQH/MB0GA1UdDgQWBBQ6Rtcn823/fxRZyheRDFpDzuBMpTAKBggqhkjO
PQQDAgNIADBFAiEAqD3Kb2rgsy6NoTk+zEgcUi/aGBCsvQDG3vML1PXN8j0CIBjj
4nDj+GmHXcnKa8wXx70Z8OZEpRQIiKDDLmcXuslp
-----END CERTIFICATE-----
`))
	tmpfile.Close()

	errorTestCases := [][]string{
		{
			"--ca-cert=" + tmpfile.Name(),
		},
		{
			"--ca-key=" + tmpfile.Name(),
		},
		{ // to make sure we're not erroring because we didn't provide a CA key along with the CA cert

			"--ca-cert=" + tmpfile.Name(),
			"--ca-key=" + tmpfile.Name(),
		},
		{
			"--cert-expiry=2160h0m0s",
		},
		{
			"--external-ca=protocol=cfssl,url=https://some.com/https/url",
		},
		{ // to make sure we're not erroring because we didn't provide a CA cert and external CA

			"--ca-cert=" + tmpfile.Name(),
			"--external-ca=protocol=cfssl,url=https://some.com/https/url",
		},
	}

	for _, args := range errorTestCases {
		cmd := newCACommand(
			test.NewFakeCli(&fakeClient{
				swarmInspectFunc: func() (swarm.Swarm, error) {
					return swarm.Swarm{
						ClusterInfo: swarm.ClusterInfo{
							TLSInfo: swarm.TLSInfo{
								TrustRoot: "root",
							},
						},
					}, nil
				},
			}))
		assert.Check(t, cmd.Flags().Parse(args))
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), "flag requires the `--rotate` flag to update the CA")
	}
}

func TestDisplayTrustRoot(t *testing.T) {
	buffer := new(bytes.Buffer)
	trustRoot := "trustme"
	err := displayTrustRoot(buffer, swarm.Swarm{
		ClusterInfo: swarm.ClusterInfo{
			TLSInfo: swarm.TLSInfo{TrustRoot: trustRoot},
		},
	})
	assert.NilError(t, err)
	assert.Check(t, is.Equal(trustRoot+"\n", buffer.String()))
}

func TestUpdateSwarmSpecDefaultRotate(t *testing.T) {
	spec := swarmSpecWithFullCAConfig()
	flags := newCACommand(nil).Flags()
	updateSwarmSpec(spec, flags, caOptions{})

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.ForceRotate = 2
	expected.CAConfig.SigningCACert = ""
	expected.CAConfig.SigningCAKey = ""
	assert.Check(t, is.DeepEqual(expected, spec))
}

func TestUpdateSwarmSpecPartial(t *testing.T) {
	spec := swarmSpecWithFullCAConfig()
	flags := newCACommand(nil).Flags()
	updateSwarmSpec(spec, flags, caOptions{
		rootCACert: PEMFile{contents: "cacert"},
	})

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.SigningCACert = "cacert"
	assert.Check(t, is.DeepEqual(expected, spec))
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
	assert.Check(t, is.DeepEqual(expected, spec))
}
