package command

import (
	"context"
	"crypto/x509"
	"os"
	"runtime"
	"testing"

	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
	"github.com/gotestyourself/gotestyourself/env"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/pkg/errors"
)

func TestNewAPIClientFromFlags(t *testing.T) {
	host := "unix://path"
	if runtime.GOOS == "windows" {
		host = "npipe://./"
	}
	opts := &flags.CommonOptions{Hosts: []string{host}}
	configFile := &configfile.ConfigFile{
		HTTPHeaders: map[string]string{
			"My-Header": "Custom-Value",
		},
	}
	apiclient, err := NewAPIClientFromFlags(opts, configFile)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(host, apiclient.DaemonHost()))

	expectedHeaders := map[string]string{
		"My-Header":  "Custom-Value",
		"User-Agent": UserAgent(),
	}
	assert.Check(t, is.DeepEqual(expectedHeaders, apiclient.(*client.Client).CustomHTTPHeaders()))
	assert.Check(t, is.Equal(api.DefaultVersion, apiclient.ClientVersion()))
}

func TestNewAPIClientFromFlagsWithAPIVersionFromEnv(t *testing.T) {
	customVersion := "v3.3.3"
	defer env.Patch(t, "DOCKER_API_VERSION", customVersion)()

	opts := &flags.CommonOptions{}
	configFile := &configfile.ConfigFile{}
	apiclient, err := NewAPIClientFromFlags(opts, configFile)
	assert.NilError(t, err)
	assert.Check(t, is.Equal(customVersion, apiclient.ClientVersion()))
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
		t.Run(testcase.doc, func(t *testing.T) {
			apiclient := &fakeClient{
				pingFunc: testcase.pingFunc,
				version:  defaultVersion,
			}

			cli := &DockerCli{client: apiclient}
			cli.initializeFromClient()
			assert.Check(t, is.DeepEqual(testcase.expectedServer, cli.serverInfo))
			assert.Check(t, is.Equal(testcase.negotiated, apiclient.negotiated))
		})
	}
}

func TestExperimentalCLI(t *testing.T) {
	defaultVersion := "v1.55"

	var testcases = []struct {
		doc                     string
		configfile              string
		expectedExperimentalCLI bool
	}{
		{
			doc:                     "default",
			configfile:              `{}`,
			expectedExperimentalCLI: false,
		},
		{
			doc: "experimental",
			configfile: `{
	"experimental": "enabled"
}`,
			expectedExperimentalCLI: true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			dir := fs.NewDir(t, testcase.doc, fs.WithFile("config.json", testcase.configfile))
			defer dir.Remove()
			apiclient := &fakeClient{
				version: defaultVersion,
			}

			cli := &DockerCli{client: apiclient, err: os.Stderr}
			cliconfig.SetDir(dir.Path())
			err := cli.Initialize(flags.NewClientOptions())
			assert.NilError(t, err)
			assert.Check(t, is.Equal(testcase.expectedExperimentalCLI, cli.ClientInfo().HasExperimental))
		})
	}
}

func TestOrchestratorSwitch(t *testing.T) {
	defaultVersion := "v0.00"

	var testcases = []struct {
		doc                  string
		configfile           string
		envOrchestrator      string
		flagOrchestrator     string
		expectedOrchestrator string
		expectedKubernetes   bool
		expectedSwarm        bool
	}{
		{
			doc: "default",
			configfile: `{
			}`,
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
		{
			doc: "kubernetesConfigFile",
			configfile: `{
				"orchestrator": "kubernetes"
			}`,
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc: "kubernetesEnv",
			configfile: `{
			}`,
			envOrchestrator:      "kubernetes",
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc: "kubernetesFlag",
			configfile: `{
			}`,
			flagOrchestrator:     "kubernetes",
			expectedOrchestrator: "kubernetes",
			expectedKubernetes:   true,
			expectedSwarm:        false,
		},
		{
			doc: "allOrchestratorFlag",
			configfile: `{
			}`,
			flagOrchestrator:     "all",
			expectedOrchestrator: "all",
			expectedKubernetes:   true,
			expectedSwarm:        true,
		},
		{
			doc: "envOverridesConfigFile",
			configfile: `{
				"orchestrator": "kubernetes"
			}`,
			envOrchestrator:      "swarm",
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
		{
			doc: "flagOverridesEnv",
			configfile: `{
			}`,
			envOrchestrator:      "kubernetes",
			flagOrchestrator:     "swarm",
			expectedOrchestrator: "swarm",
			expectedKubernetes:   false,
			expectedSwarm:        true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			dir := fs.NewDir(t, testcase.doc, fs.WithFile("config.json", testcase.configfile))
			defer dir.Remove()
			apiclient := &fakeClient{
				version: defaultVersion,
			}
			if testcase.envOrchestrator != "" {
				defer env.Patch(t, "DOCKER_ORCHESTRATOR", testcase.envOrchestrator)()
			}

			cli := &DockerCli{client: apiclient, err: os.Stderr}
			cliconfig.SetDir(dir.Path())
			options := flags.NewClientOptions()
			if testcase.flagOrchestrator != "" {
				options.Common.Orchestrator = testcase.flagOrchestrator
			}
			err := cli.Initialize(options)
			assert.NilError(t, err)
			assert.Check(t, is.Equal(testcase.expectedKubernetes, cli.ClientInfo().HasKubernetes()))
			assert.Check(t, is.Equal(testcase.expectedSwarm, cli.ClientInfo().HasSwarm()))
			assert.Check(t, is.Equal(testcase.expectedOrchestrator, string(cli.ClientInfo().Orchestrator)))
		})
	}
}

func TestGetClientWithPassword(t *testing.T) {
	expected := "password"

	var testcases = []struct {
		doc             string
		password        string
		retrieverErr    error
		retrieverGiveup bool
		newClientErr    error
		expectedErr     string
	}{
		{
			doc:      "successful connect",
			password: expected,
		},
		{
			doc:             "password retriever exhausted",
			retrieverGiveup: true,
			retrieverErr:    errors.New("failed"),
			expectedErr:     "private key is encrypted, but could not get passphrase",
		},
		{
			doc:          "password retriever error",
			retrieverErr: errors.New("failed"),
			expectedErr:  "failed",
		},
		{
			doc:          "newClient error",
			newClientErr: errors.New("failed to connect"),
			expectedErr:  "failed to connect",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.doc, func(t *testing.T) {
			passRetriever := func(_, _ string, _ bool, attempts int) (passphrase string, giveup bool, err error) {
				// Always return an invalid pass first to test iteration
				switch attempts {
				case 0:
					return "something else", false, nil
				default:
					return testcase.password, testcase.retrieverGiveup, testcase.retrieverErr
				}
			}

			newClient := func(currentPassword string) (client.APIClient, error) {
				if testcase.newClientErr != nil {
					return nil, testcase.newClientErr
				}
				if currentPassword == expected {
					return &client.Client{}, nil
				}
				return &client.Client{}, x509.IncorrectPasswordError
			}

			_, err := getClientWithPassword(passRetriever, newClient)
			if testcase.expectedErr != "" {
				assert.ErrorContains(t, err, testcase.expectedErr)
				return
			}

			assert.NilError(t, err)
		})
	}
}
