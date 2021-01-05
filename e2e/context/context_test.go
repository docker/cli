package context

import (
	"io/ioutil"
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/golden"
	"gotest.tools/v3/icmd"
)

func TestContextList(t *testing.T) {
	cmd := icmd.Command("docker", "context", "ls")
	cmd.Env = append(cmd.Env,
		"DOCKER_CONFIG=./testdata/test-dockerconfig",
		"KUBECONFIG=./testdata/test-kubeconfig",
	)
	result := icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		Err:      icmd.None,
		ExitCode: 0,
	})
	golden.Assert(t, result.Stdout(), "context-ls.golden")
}

func TestContextImportNoTLS(t *testing.T) {
	d, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(d)
	}()
	cmd := icmd.Command("docker", "context", "import", "remote", "./testdata/test-dockerconfig.tar")
	cmd.Env = append(cmd.Env,
		"DOCKER_CONFIG="+d,
	)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd = icmd.Command("docker", "context", "ls")
	cmd.Env = append(cmd.Env,
		"DOCKER_CONFIG="+d,
		"KUBECONFIG=./testdata/test-kubeconfig", // Allows reuse of context-ls.golden
	)
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "context-ls.golden")
}

func TestContextImportTLS(t *testing.T) {
	d, _ := ioutil.TempDir("", "")
	defer func() {
		os.RemoveAll(d)
	}()
	cmd := icmd.Command("docker", "context", "import", "test", "./testdata/test-dockerconfig-tls.tar")
	cmd.Env = append(cmd.Env,
		"DOCKER_CONFIG="+d,
	)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd = icmd.Command("docker", "context", "ls")
	cmd.Env = append(cmd.Env,
		"DOCKER_CONFIG="+d,
	)
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	golden.Assert(t, result.Stdout(), "context-ls-tls.golden")

	b, err := ioutil.ReadFile(d + "/contexts/tls/9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08/kubernetes/key.pem")
	assert.NilError(t, err)
	assert.Equal(t, string(b), `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEArQk77K5sgrQYY6HiQ1y7AC+67HrRB36oEvR+Fq60RsFcc3cZ
xAvMkRSBPjQyskjdYY7kfykGHhfJxGKopb3cDJx3eDBxjgAniwnnOMmHVWbf8Eik
o0sNxkgzQPGq83nL3QvVxm3xgqe4nlTdR/Swoq6Pv0oaVYvPPMnaZIF89SJ/wlNT
myCs6Uq00dICi20II+M2Nw9b+EVEK4ENl+SlrsK7iuoBIh/H0ZghxOthO9J/HeBb
hmM4wcs1OonhPDYKHEaChYA7/Q3/8OBp3bAdlQJ1ziyP3ROAKHL2NwwkGZ8o8HP8
u0ex/NAb8w5J5WNePqYQd/sqfisfNpA5VIKcEQIDAQABAoIBABLo4W2aGi2mdMve
kxV9esoobSsOuO0ywDdiFK1x5i2dT/cmWuB70Z1BOmaL2cZ2BAt3TC1BVHPRcbFO
ftOuDfAq4Tt3P9Ge3rNpH6WrEGka1voxVhyqRRUYKtG8F0yIUOkVNAV9WllG7vwO
ligY63y7yuXCuWID51/jR0SYiglXz6G4gcJKFXtugXXiLUIg08GVWkwOsrACC+hR
mhcHly1926VhN5+ozjNU/GZ1LaTuK6erBZakH5bqlN97s5rrk0ZRwk/JtnkoRRdI
cq0918Za2vqGDHZ3MqLttL52YfDXPIEJPwlFdvC/+sXK2NhUB/xY4yuliU3sY0sf
XsIvIWECgYEAwD8AnZI0hnGv8hc6zJppHFRwhrtLZ+09SJwPv5Y4wxuuk5dzNkpf
xCNo5hjSVYA1MMmWG8p/sEXo2IyCT8sWDNCn9kieTXihxRxbj88Y2qA5O4N46Zy4
kPngjkP5PPDMkwaQQgUr9LvlWS7P6OJkH18ZN8s3QhMaKcHu9FFT44UCgYEA5mte
mMSDf9hUK3IK+yrGX62qc2H+ecXN3Zf3nehyiz+dX4ZXhBwBkwJ/mHvuAZPfoFUN
Xg6cdyWFJg9ynm45JXnDjmYPGmFLn0mP3Mje/+SbbW2fdFWHJW/maqj4uUqqgQd+
pGNzKXq34MzDrpsqIJ7AHu3LYVMOoLAVqC7LXh0CgYEAnLF9ZfFqQH7fgvouIeBl
dgLZKOf2AUJcJheVunnN0DF67K+P55tdTTfzY0CuB6SVNivI3uQBiYKh1AdKm5ET
auSTUmlEJi8B4/BGLQQG5QOdQoXZgsgLo5cX0b1To7k9dUTvRfCDMFoKCNPgAJiu
NOfFXTWU15VMSObaRmcXciUCgYEA5e1cXwsxwUAodZX+eTXs8ArHHQ47Nl55GFeN
wufybRuUuX7AE9cyhvUmSA3aqX5a144noaTo40fwftNJZ+jLY6cGyjDzfzp5kMCC
KynSxPzlUCPkytyR2Hy6K9LjJ1rnm4vUBswqXcjUdiE+Xxz8w8JGKlbV7Q9JeHVd
lw7i5s0CgYAn9T9ySI3xCbrUa/XV/ZY2hopUdH5CDPeTd2eH+L+lctkD9nlzLrpj
qij+jaEUweymNx0uttgv02J3DYcIIvVq3RNAwORy5Mp9KasHmjbW2xq+HAq5yFOO
1ma82F5zeUl+bKqjMRCY8IVZ349VxRZtb2RVVEKyVswb7HmKp6gGbA==
-----END RSA PRIVATE KEY-----
`)
}
