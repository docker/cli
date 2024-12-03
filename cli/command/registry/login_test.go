package registry

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/docker/cli/cli/command"
	configtypes "github.com/docker/cli/cli/config/types"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/internal/test"
	registrytypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/client"
	"github.com/docker/docker/registry"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
	"gotest.tools/v3/fs"
)

const (
	unknownUser     = "userunknownError"
	errUnknownUser  = "UNKNOWN_ERR"
	expiredPassword = "I_M_EXPIRED"
	useToken        = "I_M_TOKEN"
)

type fakeClient struct {
	client.Client
}

func (c *fakeClient) Info(context.Context) (system.Info, error) {
	return system.Info{}, nil
}

func (c *fakeClient) RegistryLogin(_ context.Context, auth registrytypes.AuthConfig) (registrytypes.AuthenticateOKBody, error) {
	if auth.Password == expiredPassword {
		return registrytypes.AuthenticateOKBody{}, errors.New("Invalid Username or Password")
	}
	if auth.Password == useToken {
		return registrytypes.AuthenticateOKBody{
			IdentityToken: auth.Password,
		}, nil
	}
	if auth.Username == unknownUser {
		return registrytypes.AuthenticateOKBody{}, errors.New(errUnknownUser)
	}
	return registrytypes.AuthenticateOKBody{}, nil
}

func TestLoginWithCredStoreCreds(t *testing.T) {
	testCases := []struct {
		inputAuthConfig registrytypes.AuthConfig
		expectedErr     string
		expectedMsg     string
		expectedErrMsg  string
	}{
		{
			inputAuthConfig: registrytypes.AuthConfig{},
			expectedMsg:     "Authenticating with existing credentials...\n",
		},
		{
			inputAuthConfig: registrytypes.AuthConfig{
				Username: unknownUser,
			},
			expectedErr:    errUnknownUser,
			expectedMsg:    "Authenticating with existing credentials...\n",
			expectedErrMsg: fmt.Sprintf("Login did not succeed, error: %s\n", errUnknownUser),
		},
	}
	ctx := context.Background()
	cli := test.NewFakeCli(&fakeClient{})
	cli.ConfigFile().Filename = filepath.Join(t.TempDir(), "config.json")
	for _, tc := range testCases {
		_, err := loginWithStoredCredentials(ctx, cli, tc.inputAuthConfig)
		if tc.expectedErrMsg != "" {
			assert.Check(t, is.Error(err, tc.expectedErr))
		} else {
			assert.NilError(t, err)
		}
		assert.Check(t, is.Equal(tc.expectedMsg, cli.OutBuffer().String()))
		assert.Check(t, is.Equal(tc.expectedErrMsg, cli.ErrBuffer().String()))
		cli.ErrBuffer().Reset()
		cli.OutBuffer().Reset()
	}
}

func TestRunLogin(t *testing.T) {
	testCases := []struct {
		doc                 string
		priorCredentials    map[string]configtypes.AuthConfig
		input               loginOptions
		expectedCredentials map[string]configtypes.AuthConfig
		expectedErr         string
	}{
		{
			doc: "valid auth from store",
			priorCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "my-username",
					Password:      "a-password",
					ServerAddress: "reg1",
				},
			},
			input: loginOptions{
				serverAddress: "reg1",
			},
			expectedCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "my-username",
					Password:      "a-password",
					ServerAddress: "reg1",
				},
			},
		},
		{
			doc: "expired auth from store",
			priorCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "my-username",
					Password:      expiredPassword,
					ServerAddress: "reg1",
				},
			},
			input: loginOptions{
				serverAddress: "reg1",
			},
			expectedErr: "Error: Cannot perform an interactive login from a non TTY device",
		},
		{
			doc:              "store valid username and password",
			priorCredentials: map[string]configtypes.AuthConfig{},
			input: loginOptions{
				serverAddress: "reg1",
				user:          "my-username",
				password:      "p2",
			},
			expectedCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "my-username",
					Password:      "p2",
					ServerAddress: "reg1",
				},
			},
		},
		{
			doc: "unknown user w/ prior credentials",
			priorCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "my-username",
					Password:      "a-password",
					ServerAddress: "reg1",
				},
			},
			input: loginOptions{
				serverAddress: "reg1",
				user:          unknownUser,
				password:      "a-password",
			},
			expectedErr: errUnknownUser,
			expectedCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "a-password",
					Password:      "a-password",
					ServerAddress: "reg1",
				},
			},
		},
		{
			doc:              "unknown user w/o prior credentials",
			priorCredentials: map[string]configtypes.AuthConfig{},
			input: loginOptions{
				serverAddress: "reg1",
				user:          unknownUser,
				password:      "a-password",
			},
			expectedErr:         errUnknownUser,
			expectedCredentials: map[string]configtypes.AuthConfig{},
		},
		{
			doc:              "store valid token",
			priorCredentials: map[string]configtypes.AuthConfig{},
			input: loginOptions{
				serverAddress: "reg1",
				user:          "my-username",
				password:      useToken,
			},
			expectedCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "my-username",
					IdentityToken: useToken,
					ServerAddress: "reg1",
				},
			},
		},
		{
			doc: "valid token from store",
			priorCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "my-username",
					Password:      useToken,
					ServerAddress: "reg1",
				},
			},
			input: loginOptions{
				serverAddress: "reg1",
			},
			expectedCredentials: map[string]configtypes.AuthConfig{
				"reg1": {
					Username:      "my-username",
					IdentityToken: useToken,
					ServerAddress: "reg1",
				},
			},
		},
		{
			doc:              "no registry specified defaults to index server",
			priorCredentials: map[string]configtypes.AuthConfig{},
			input: loginOptions{
				user:     "my-username",
				password: "my-password",
			},
			expectedCredentials: map[string]configtypes.AuthConfig{
				registry.IndexServer: {
					Username:      "my-username",
					Password:      "my-password",
					ServerAddress: registry.IndexServer,
				},
			},
		},
		{
			doc:              "registry-1.docker.io",
			priorCredentials: map[string]configtypes.AuthConfig{},
			input: loginOptions{
				serverAddress: "registry-1.docker.io",
				user:          "my-username",
				password:      "my-password",
			},
			expectedCredentials: map[string]configtypes.AuthConfig{
				"registry-1.docker.io": {
					Username:      "my-username",
					Password:      "my-password",
					ServerAddress: "registry-1.docker.io",
				},
			},
		},
		// Regression test for https://github.com/docker/cli/issues/5382
		{
			doc:              "sanitizes server address to remove repo",
			priorCredentials: map[string]configtypes.AuthConfig{},
			input: loginOptions{
				serverAddress: "registry-1.docker.io/bork/test",
				user:          "my-username",
				password:      "a-password",
			},
			expectedCredentials: map[string]configtypes.AuthConfig{
				"registry-1.docker.io": {
					Username:      "my-username",
					Password:      "a-password",
					ServerAddress: "registry-1.docker.io",
				},
			},
		},
		// Regression test for https://github.com/docker/cli/issues/5382
		{
			doc: "updates credential if server address includes repo",
			priorCredentials: map[string]configtypes.AuthConfig{
				"registry-1.docker.io": {
					Username:      "my-username",
					Password:      "a-password",
					ServerAddress: "registry-1.docker.io",
				},
			},
			input: loginOptions{
				serverAddress: "registry-1.docker.io/bork/test",
				user:          "my-username",
				password:      "new-password",
			},
			expectedCredentials: map[string]configtypes.AuthConfig{
				"registry-1.docker.io": {
					Username:      "my-username",
					Password:      "new-password",
					ServerAddress: "registry-1.docker.io",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.doc, func(t *testing.T) {
			tmpFile := fs.NewFile(t, "test-run-login")
			defer tmpFile.Remove()
			cli := test.NewFakeCli(&fakeClient{})
			configfile := cli.ConfigFile()
			configfile.Filename = tmpFile.Path()

			for _, priorCred := range tc.priorCredentials {
				assert.NilError(t, configfile.GetCredentialsStore(priorCred.ServerAddress).Store(priorCred))
			}
			storedCreds, err := configfile.GetAllCredentials()
			assert.NilError(t, err)
			assert.DeepEqual(t, storedCreds, tc.priorCredentials)

			loginErr := runLogin(context.Background(), cli, tc.input)
			if tc.expectedErr != "" {
				assert.Error(t, loginErr, tc.expectedErr)
				return
			}
			assert.NilError(t, loginErr)

			outputCreds, err := configfile.GetAllCredentials()
			assert.Check(t, err)
			assert.DeepEqual(t, outputCreds, tc.expectedCredentials)
		})
	}
}

func TestLoginNonInteractive(t *testing.T) {
	t.Run("no prior credentials", func(t *testing.T) {
		testCases := []struct {
			doc         string
			username    bool
			password    bool
			expectedErr string
		}{
			{
				doc:      "success - w/ user w/ password",
				username: true,
				password: true,
			},
			{
				doc:         "error - w/o user w/o pass ",
				username:    false,
				password:    false,
				expectedErr: "Error: Cannot perform an interactive login from a non TTY device",
			},
			{
				doc:         "error - w/ user w/o pass",
				username:    true,
				password:    false,
				expectedErr: "Error: Cannot perform an interactive login from a non TTY device",
			},
			{
				doc:         "error - w/o user w/ pass",
				username:    false,
				password:    true,
				expectedErr: "Error: Cannot perform an interactive login from a non TTY device",
			},
		}

		// "" meaning default registry
		registries := []string{"", "my-registry.com"}

		for _, registryAddr := range registries {
			for _, tc := range testCases {
				t.Run(tc.doc, func(t *testing.T) {
					tmpFile := fs.NewFile(t, "test-run-login")
					defer tmpFile.Remove()
					cli := test.NewFakeCli(&fakeClient{})
					cfg := cli.ConfigFile()
					cfg.Filename = tmpFile.Path()
					options := loginOptions{
						serverAddress: registryAddr,
					}
					if tc.username {
						options.user = "my-username"
					}
					if tc.password {
						options.password = "my-password"
					}

					loginErr := runLogin(context.Background(), cli, options)
					if tc.expectedErr != "" {
						assert.Error(t, loginErr, tc.expectedErr)
						return
					}
					assert.NilError(t, loginErr)
				})
			}
		}
	})

	t.Run("w/ prior credentials", func(t *testing.T) {
		testCases := []struct {
			doc         string
			username    bool
			password    bool
			expectedErr string
		}{
			{
				doc:      "success - w/ user w/ password",
				username: true,
				password: true,
			},
			{
				doc:      "success - w/o user w/o pass ",
				username: false,
				password: false,
			},
			{
				doc:         "error - w/ user w/o pass",
				username:    true,
				password:    false,
				expectedErr: "Error: Cannot perform an interactive login from a non TTY device",
			},
			{
				doc:         "error - w/o user w/ pass",
				username:    false,
				password:    true,
				expectedErr: "Error: Cannot perform an interactive login from a non TTY device",
			},
		}

		// "" meaning default registry
		registries := []string{"", "my-registry.com"}

		for _, registryAddr := range registries {
			for _, tc := range testCases {
				t.Run(tc.doc, func(t *testing.T) {
					tmpFile := fs.NewFile(t, "test-run-login")
					defer tmpFile.Remove()
					cli := test.NewFakeCli(&fakeClient{})
					cfg := cli.ConfigFile()
					cfg.Filename = tmpFile.Path()
					serverAddress := registryAddr
					if serverAddress == "" {
						serverAddress = "https://index.docker.io/v1/"
					}
					assert.NilError(t, cfg.GetCredentialsStore(serverAddress).Store(configtypes.AuthConfig{
						Username:      "my-username",
						Password:      "my-password",
						ServerAddress: serverAddress,
					}))

					options := loginOptions{
						serverAddress: registryAddr,
					}
					if tc.username {
						options.user = "my-username"
					}
					if tc.password {
						options.password = "my-password"
					}

					loginErr := runLogin(context.Background(), cli, options)
					if tc.expectedErr != "" {
						assert.Error(t, loginErr, tc.expectedErr)
						return
					}
					assert.NilError(t, loginErr)
				})
			}
		}
	})
}

func TestLoginTermination(t *testing.T) {
	p, tty, err := pty.Open()
	assert.NilError(t, err)

	t.Cleanup(func() {
		_ = tty.Close()
		_ = p.Close()
	})

	cli := test.NewFakeCli(&fakeClient{}, func(fc *test.FakeCli) {
		fc.SetOut(streams.NewOut(tty))
		fc.SetIn(streams.NewIn(tty))
	})
	tmpFile := fs.NewFile(t, "test-login-termination")
	defer tmpFile.Remove()

	configFile := cli.ConfigFile()
	configFile.Filename = tmpFile.Path()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	runErr := make(chan error)
	go func() {
		runErr <- runLogin(ctx, cli, loginOptions{
			user: "test-user",
		})
	}()

	// Let the prompt get canceled by the context
	cancel()

	select {
	case <-time.After(1 * time.Second):
		t.Fatal("timed out after 1 second. `runLogin` did not return")
	case err := <-runErr:
		assert.ErrorIs(t, err, command.ErrPromptTerminated)
	}
}

func TestIsOauthLoginDisabled(t *testing.T) {
	testCases := []struct {
		envVar   string
		disabled bool
	}{
		{
			envVar:   "",
			disabled: false,
		},
		{
			envVar:   "bork",
			disabled: false,
		},
		{
			envVar:   "0",
			disabled: false,
		},
		{
			envVar:   "false",
			disabled: false,
		},
		{
			envVar:   "true",
			disabled: true,
		},
		{
			envVar:   "TRUE",
			disabled: true,
		},
		{
			envVar:   "1",
			disabled: true,
		},
	}

	for _, tc := range testCases {
		t.Setenv(OauthLoginEscapeHatchEnvVar, tc.envVar)

		disabled := isOauthLoginDisabled()

		assert.Equal(t, disabled, tc.disabled)
	}
}
