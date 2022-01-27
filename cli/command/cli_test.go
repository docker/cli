package command

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"
)

func TestNewAPIClientFromFlags(t *testing.T) {
	host := "unix://path"
	if runtime.GOOS == "windows" {
		host = "npipe://./"
	}
	opts := &flags.CommonOptions{Hosts: []string{host}}
	apiClient, err := NewAPIClientFromFlags(opts, &configfile.ConfigFile{})
	assert.NilError(t, err)
	assert.Equal(t, apiClient.DaemonHost(), host)
	assert.Equal(t, apiClient.ClientVersion(), api.DefaultVersion)
}

func TestNewAPIClientFromFlagsForDefaultSchema(t *testing.T) {
	host := ":2375"
	slug := "tcp://localhost"
	if runtime.GOOS == "windows" {
		slug = "tcp://127.0.0.1"
	}
	opts := &flags.CommonOptions{Hosts: []string{host}}
	apiClient, err := NewAPIClientFromFlags(opts, &configfile.ConfigFile{})
	assert.NilError(t, err)
	assert.Equal(t, apiClient.DaemonHost(), slug+host)
	assert.Equal(t, apiClient.ClientVersion(), api.DefaultVersion)
}

func TestNewAPIClientFromFlagsWithCustomHeaders(t *testing.T) {
	var received map[string]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = map[string]string{
			"My-Header":  r.Header.Get("My-Header"),
			"User-Agent": r.Header.Get("User-Agent"),
		}
		_, _ = w.Write([]byte("OK"))
	}))
	defer ts.Close()
	host := strings.Replace(ts.URL, "http://", "tcp://", 1)
	opts := &flags.CommonOptions{Hosts: []string{host}}
	configFile := &configfile.ConfigFile{
		HTTPHeaders: map[string]string{
			"My-Header": "Custom-Value",
		},
	}

	apiClient, err := NewAPIClientFromFlags(opts, configFile)
	assert.NilError(t, err)
	assert.Equal(t, apiClient.DaemonHost(), host)
	assert.Equal(t, apiClient.ClientVersion(), api.DefaultVersion)

	// verify User-Agent is not appended to the configfile. see https://github.com/docker/cli/pull/2756
	assert.DeepEqual(t, configFile.HTTPHeaders, map[string]string{"My-Header": "Custom-Value"})

	expectedHeaders := map[string]string{
		"My-Header":  "Custom-Value",
		"User-Agent": UserAgent(),
	}
	_, err = apiClient.Ping(context.Background())
	assert.NilError(t, err)
	assert.DeepEqual(t, received, expectedHeaders)
}

func TestNewAPIClientFromFlagsWithAPIVersionFromEnv(t *testing.T) {
	customVersion := "v3.3.3"
	defer env.Patch(t, "DOCKER_API_VERSION", customVersion)()
	defer env.Patch(t, "DOCKER_HOST", ":2375")()

	opts := &flags.CommonOptions{}
	configFile := &configfile.ConfigFile{}
	apiclient, err := NewAPIClientFromFlags(opts, configFile)
	assert.NilError(t, err)
	assert.Equal(t, apiclient.ClientVersion(), customVersion)
}

type fakeClient struct {
	client.Client
	pingFunc   func() (types.Ping, error)
	version    string
	negotiated bool
}

func (c *fakeClient) Ping(_ context.Context) (types.Ping, error) {
	return c.pingFunc()
}

func (c *fakeClient) ClientVersion() string {
	return c.version
}

func (c *fakeClient) NegotiateAPIVersionPing(types.Ping) {
	c.negotiated = true
}

func TestInitializeFromClient(t *testing.T) {
	defaultVersion := "v1.55"

	var testcases = []struct {
		doc            string
		pingFunc       func() (types.Ping, error)
		expectedServer ServerInfo
		negotiated     bool
	}{
		{
			doc: "successful ping",
			pingFunc: func() (types.Ping, error) {
				return types.Ping{Experimental: true, OSType: "linux", APIVersion: "v1.30"}, nil
			},
			expectedServer: ServerInfo{HasExperimental: true, OSType: "linux"},
			negotiated:     true,
		},
		{
			doc: "failed ping, no API version",
			pingFunc: func() (types.Ping, error) {
				return types.Ping{}, errors.New("failed")
			},
			expectedServer: ServerInfo{HasExperimental: true},
		},
		{
			doc: "failed ping, with API version",
			pingFunc: func() (types.Ping, error) {
				return types.Ping{APIVersion: "v1.33"}, errors.New("failed")
			},
			expectedServer: ServerInfo{HasExperimental: true},
			negotiated:     true,
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.doc, func(t *testing.T) {
			apiclient := &fakeClient{
				pingFunc: testcase.pingFunc,
				version:  defaultVersion,
			}

			cli := &DockerCli{client: apiclient}
			cli.initializeFromClient()
			assert.DeepEqual(t, cli.serverInfo, testcase.expectedServer)
			assert.Equal(t, apiclient.negotiated, testcase.negotiated)
		})
	}
}

// The CLI no longer disables/hides experimental CLI features, however, we need
// to verify that existing configuration files do not break
func TestExperimentalCLI(t *testing.T) {
	defaultVersion := "v1.55"

	var testcases = []struct {
		doc        string
		configfile string
	}{
		{
			doc:        "default",
			configfile: `{}`,
		},
		{
			doc: "experimental",
			configfile: `{
	"experimental": "enabled"
}`,
		},
	}

	for _, testcase := range testcases {
		testcase := testcase
		t.Run(testcase.doc, func(t *testing.T) {
			dir := fs.NewDir(t, testcase.doc, fs.WithFile("config.json", testcase.configfile))
			defer dir.Remove()
			apiclient := &fakeClient{
				version: defaultVersion,
				pingFunc: func() (types.Ping, error) {
					return types.Ping{Experimental: true, OSType: "linux", APIVersion: defaultVersion}, nil
				},
			}

			cli := &DockerCli{client: apiclient, err: os.Stderr}
			cliconfig.SetDir(dir.Path())
			err := cli.Initialize(flags.NewClientOptions())
			assert.NilError(t, err)
			// For backward-compatibility, HasExperimental will always be "true"
			assert.Equal(t, cli.ClientInfo().HasExperimental, true)
		})
	}
}

func TestNewDockerCliAndOperators(t *testing.T) {
	// Test default operations and also overriding default ones
	cli, err := NewDockerCli(
		WithContentTrust(true),
	)
	assert.NilError(t, err)
	// Check streams are initialized
	assert.Check(t, cli.In() != nil)
	assert.Check(t, cli.Out() != nil)
	assert.Check(t, cli.Err() != nil)
	assert.Equal(t, cli.ContentTrustEnabled(), true)

	// Apply can modify a dockerCli after construction
	inbuf := bytes.NewBuffer([]byte("input"))
	outbuf := bytes.NewBuffer(nil)
	errbuf := bytes.NewBuffer(nil)
	err = cli.Apply(
		WithInputStream(ioutil.NopCloser(inbuf)),
		WithOutputStream(outbuf),
		WithErrorStream(errbuf),
	)
	assert.NilError(t, err)
	// Check input stream
	inputStream, err := ioutil.ReadAll(cli.In())
	assert.NilError(t, err)
	assert.Equal(t, string(inputStream), "input")
	// Check output stream
	fmt.Fprintf(cli.Out(), "output")
	outputStream, err := ioutil.ReadAll(outbuf)
	assert.NilError(t, err)
	assert.Equal(t, string(outputStream), "output")
	// Check error stream
	fmt.Fprintf(cli.Err(), "error")
	errStream, err := ioutil.ReadAll(errbuf)
	assert.NilError(t, err)
	assert.Equal(t, string(errStream), "error")
}

func TestInitializeShouldAlwaysCreateTheContextStore(t *testing.T) {
	cli, err := NewDockerCli()
	assert.NilError(t, err)
	assert.NilError(t, cli.Initialize(flags.NewClientOptions(), WithInitializeClient(func(cli *DockerCli) (client.APIClient, error) {
		return client.NewClientWithOpts()
	})))
	assert.Check(t, cli.ContextStore() != nil)
}
