package swarm

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/docker/cli/internal/test"
	"github.com/docker/docker/api/types/swarm"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const (
	cert = `
-----BEGIN CERTIFICATE-----
MIIBuDCCAV4CCQDOqUYOWdqMdjAKBggqhkjOPQQDAzBjMQswCQYDVQQGEwJVUzEL
MAkGA1UECAwCQ0ExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xDzANBgNVBAoMBkRv
Y2tlcjEPMA0GA1UECwwGRG9ja2VyMQ0wCwYDVQQDDARUZXN0MCAXDTE4MDcwMjIx
MjkxOFoYDzMwMTcxMTAyMjEyOTE4WjBjMQswCQYDVQQGEwJVUzELMAkGA1UECAwC
Q0ExFjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xDzANBgNVBAoMBkRvY2tlcjEPMA0G
A1UECwwGRG9ja2VyMQ0wCwYDVQQDDARUZXN0MFkwEwYHKoZIzj0CAQYIKoZIzj0D
AQcDQgAEgvvZl5Vqpr1e+g5IhoU6TZHgRau+BZETVFTmqyWYajA/mooRQ1MZTozu
s9ZZZA8tzUhIqS36gsFuyIZ4YiAlyjAKBggqhkjOPQQDAwNIADBFAiBQ7pCPQrj8
8zaItMf0pk8j1NU5XrFqFEZICzvjzUJQBAIhAKq2gFwoTn8KH+cAAXZpAGJPmOsT
zsBT8gBAOHhNA6/2
-----END CERTIFICATE-----`
	key = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEICyheZpw70pbgO4hEuwhZTETWyTpNJmJ3TyFaWT6WTRkoAoGCCqGSM49
AwEHoUQDQgAEgvvZl5Vqpr1e+g5IhoU6TZHgRau+BZETVFTmqyWYajA/mooRQ1MZ
Tozus9ZZZA8tzUhIqS36gsFuyIZ4YiAlyg==
-----END EC PRIVATE KEY-----`
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

type invalidCATestCases struct {
	args     []string
	errorMsg string
}

func writeFile(data string) (string, error) {
	tmpfile, err := ioutil.TempFile("", "testfile")
	if err != nil {
		return "", err
	}
	_, err = tmpfile.Write([]byte(data))
	if err != nil {
		return "", err
	}
	tmpfile.Close()
	return tmpfile.Name(), nil
}

func TestDisplayTrustRootInvalidFlags(t *testing.T) {
	// we need an actual PEMfile to test
	tmpfile, err := writeFile(cert)
	assert.NilError(t, err)
	defer os.Remove(tmpfile)

	errorTestCases := []invalidCATestCases{
		{
			args:     []string{"--ca-cert=" + tmpfile},
			errorMsg: "flag requires the `--rotate` flag to update the CA",
		},
		{
			args:     []string{"--ca-key=" + tmpfile},
			errorMsg: "flag requires the `--rotate` flag to update the CA",
		},
		{ // to make sure we're not erroring because we didn't provide a CA key along with the CA cert
			args: []string{
				"--ca-cert=" + tmpfile,
				"--ca-key=" + tmpfile,
			},
			errorMsg: "flag requires the `--rotate` flag to update the CA",
		},
		{
			args:     []string{"--cert-expiry=2160h0m0s"},
			errorMsg: "flag requires the `--rotate` flag to update the CA",
		},
		{
			args:     []string{"--external-ca=protocol=cfssl,url=https://some.com/https/url"},
			errorMsg: "flag requires the `--rotate` flag to update the CA",
		},
		{ // to make sure we're not erroring because we didn't provide a CA cert and external CA
			args: []string{
				"--ca-cert=" + tmpfile,
				"--external-ca=protocol=cfssl,url=https://some.com/https/url",
			},
			errorMsg: "flag requires the `--rotate` flag to update the CA",
		},
		{
			args: []string{
				"--rotate",
				"--external-ca=protocol=cfssl,url=https://some.com/https/url",
			},
			errorMsg: "rotating to an external CA requires the `--ca-cert` flag to specify the external CA's cert - " +
				"to add an external CA with the current root CA certificate, use the `update` command instead",
		},
		{
			args: []string{
				"--rotate",
				"--ca-cert=" + tmpfile,
			},
			errorMsg: "the --ca-cert flag requires that a --ca-key flag and/or --external-ca flag be provided as well",
		},
	}

	for _, testCase := range errorTestCases {
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
		assert.Check(t, cmd.Flags().Parse(testCase.args))
		cmd.SetOutput(ioutil.Discard)
		assert.ErrorContains(t, cmd.Execute(), testCase.errorMsg)
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

type swarmUpdateRecorder struct {
	spec swarm.Spec
}

func (s *swarmUpdateRecorder) swarmUpdate(sp swarm.Spec, _ swarm.UpdateFlags) error {
	s.spec = sp
	return nil
}

func swarmInspectFuncWithFullCAConfig() (swarm.Swarm, error) {
	return swarm.Swarm{
		ClusterInfo: swarm.ClusterInfo{
			Spec: *swarmSpecWithFullCAConfig(),
		},
	}, nil
}

func TestUpdateSwarmSpecDefaultRotate(t *testing.T) {
	s := &swarmUpdateRecorder{}
	cli := test.NewFakeCli(&fakeClient{
		swarmInspectFunc: swarmInspectFuncWithFullCAConfig,
		swarmUpdateFunc:  s.swarmUpdate,
	})
	cmd := newCACommand(cli)
	cmd.SetArgs([]string{"--rotate", "--detach"})
	cmd.SetOutput(cli.OutBuffer())
	assert.NilError(t, cmd.Execute())

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.ForceRotate = 2
	expected.CAConfig.SigningCACert = ""
	expected.CAConfig.SigningCAKey = ""
	assert.Check(t, is.DeepEqual(*expected, s.spec))
}

func TestUpdateSwarmSpecCertAndKey(t *testing.T) {
	certfile, err := writeFile(cert)
	assert.NilError(t, err)
	defer os.Remove(certfile)

	keyfile, err := writeFile(key)
	assert.NilError(t, err)
	defer os.Remove(keyfile)

	s := &swarmUpdateRecorder{}
	cli := test.NewFakeCli(&fakeClient{
		swarmInspectFunc: swarmInspectFuncWithFullCAConfig,
		swarmUpdateFunc:  s.swarmUpdate,
	})
	cmd := newCACommand(cli)
	cmd.SetArgs([]string{
		"--rotate",
		"--detach",
		"--ca-cert=" + certfile,
		"--ca-key=" + keyfile,
		"--cert-expiry=3m"})
	cmd.SetOutput(cli.OutBuffer())
	assert.NilError(t, cmd.Execute())

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.SigningCACert = cert
	expected.CAConfig.SigningCAKey = key
	expected.CAConfig.NodeCertExpiry = 3 * time.Minute
	assert.Check(t, is.DeepEqual(*expected, s.spec))
}

func TestUpdateSwarmSpecCertAndExternalCA(t *testing.T) {
	certfile, err := writeFile(cert)
	assert.NilError(t, err)
	defer os.Remove(certfile)

	s := &swarmUpdateRecorder{}
	cli := test.NewFakeCli(&fakeClient{
		swarmInspectFunc: swarmInspectFuncWithFullCAConfig,
		swarmUpdateFunc:  s.swarmUpdate,
	})
	cmd := newCACommand(cli)
	cmd.SetArgs([]string{
		"--rotate",
		"--detach",
		"--ca-cert=" + certfile,
		"--external-ca=protocol=cfssl,url=https://some.external.ca"})
	cmd.SetOutput(cli.OutBuffer())
	assert.NilError(t, cmd.Execute())

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.SigningCACert = cert
	expected.CAConfig.SigningCAKey = ""
	expected.CAConfig.ExternalCAs = []*swarm.ExternalCA{
		{
			Protocol: swarm.ExternalCAProtocolCFSSL,
			URL:      "https://some.external.ca",
			CACert:   cert,
			Options:  make(map[string]string),
		},
	}
	assert.Check(t, is.DeepEqual(*expected, s.spec))
}

func TestUpdateSwarmSpecCertAndKeyAndExternalCA(t *testing.T) {
	certfile, err := writeFile(cert)
	assert.NilError(t, err)
	defer os.Remove(certfile)

	keyfile, err := writeFile(key)
	assert.NilError(t, err)
	defer os.Remove(keyfile)

	s := &swarmUpdateRecorder{}
	cli := test.NewFakeCli(&fakeClient{
		swarmInspectFunc: swarmInspectFuncWithFullCAConfig,
		swarmUpdateFunc:  s.swarmUpdate,
	})
	cmd := newCACommand(cli)
	cmd.SetArgs([]string{
		"--rotate",
		"--detach",
		"--ca-cert=" + certfile,
		"--ca-key=" + keyfile,
		"--external-ca=protocol=cfssl,url=https://some.external.ca"})
	cmd.SetOutput(cli.OutBuffer())
	assert.NilError(t, cmd.Execute())

	expected := swarmSpecWithFullCAConfig()
	expected.CAConfig.SigningCACert = cert
	expected.CAConfig.SigningCAKey = key
	expected.CAConfig.ExternalCAs = []*swarm.ExternalCA{
		{
			Protocol: swarm.ExternalCAProtocolCFSSL,
			URL:      "https://some.external.ca",
			CACert:   cert,
			Options:  make(map[string]string),
		},
	}
	assert.Check(t, is.DeepEqual(*expected, s.spec))
}
