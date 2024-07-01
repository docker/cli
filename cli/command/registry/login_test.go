package registry

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
		expectedMsg     string
		expectedErr     string
	}{
		{
			inputAuthConfig: registrytypes.AuthConfig{},
			expectedMsg:     "Authenticating with existing credentials...\n",
		},
		{
			inputAuthConfig: registrytypes.AuthConfig{
				Username: unknownUser,
			},
			expectedMsg: "Authenticating with existing credentials...\n",
			expectedErr: fmt.Sprintf("Login did not succeed, error: %s\n", errUnknownUser),
		},
	}
	ctx := context.Background()
	for _, tc := range testCases {
		cli := test.NewFakeCli(&fakeClient{})
		errBuf := new(bytes.Buffer)
		cli.SetErr(streams.NewOut(errBuf))
		loginWithCredStoreCreds(ctx, cli, &tc.inputAuthConfig)
		outputString := cli.OutBuffer().String()
		assert.Check(t, is.Equal(tc.expectedMsg, outputString))
		errorString := errBuf.String()
		assert.Check(t, is.Equal(tc.expectedErr, errorString))
	}
}

func TestRunLogin(t *testing.T) {
	const (
		storedServerAddress = "reg1"
		validUsername       = "u1"
		validPassword       = "p1"
		validPassword2      = "p2"
	)

	validAuthConfig := configtypes.AuthConfig{
		ServerAddress: storedServerAddress,
		Username:      validUsername,
		Password:      validPassword,
	}
	expiredAuthConfig := configtypes.AuthConfig{
		ServerAddress: storedServerAddress,
		Username:      validUsername,
		Password:      expiredPassword,
	}
	validIdentityToken := configtypes.AuthConfig{
		ServerAddress: storedServerAddress,
		Username:      validUsername,
		IdentityToken: useToken,
	}
	testCases := []struct {
		doc               string
		inputLoginOption  loginOptions
		inputStoredCred   *configtypes.AuthConfig
		expectedErr       string
		expectedSavedCred configtypes.AuthConfig
	}{
		{
			doc: "valid auth from store",
			inputLoginOption: loginOptions{
				serverAddress: storedServerAddress,
			},
			inputStoredCred:   &validAuthConfig,
			expectedSavedCred: validAuthConfig,
		},
		{
			doc: "expired auth",
			inputLoginOption: loginOptions{
				serverAddress: storedServerAddress,
			},
			inputStoredCred: &expiredAuthConfig,
			expectedErr:     "Error: Cannot perform an interactive login from a non TTY device",
		},
		{
			doc: "valid username and password",
			inputLoginOption: loginOptions{
				serverAddress: storedServerAddress,
				user:          validUsername,
				password:      validPassword2,
			},
			inputStoredCred: &validAuthConfig,
			expectedSavedCred: configtypes.AuthConfig{
				ServerAddress: storedServerAddress,
				Username:      validUsername,
				Password:      validPassword2,
			},
		},
		{
			doc: "unknown user",
			inputLoginOption: loginOptions{
				serverAddress: storedServerAddress,
				user:          unknownUser,
				password:      validPassword,
			},
			inputStoredCred: &validAuthConfig,
			expectedErr:     errUnknownUser,
		},
		{
			doc: "valid token",
			inputLoginOption: loginOptions{
				serverAddress: storedServerAddress,
				user:          validUsername,
				password:      useToken,
			},
			inputStoredCred:   &validIdentityToken,
			expectedSavedCred: validIdentityToken,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			tmpFile := fs.NewFile(t, "test-run-login")
			defer tmpFile.Remove()
			cli := test.NewFakeCli(&fakeClient{})
			configfile := cli.ConfigFile()
			configfile.Filename = tmpFile.Path()

			if tc.inputStoredCred != nil {
				cred := *tc.inputStoredCred
				assert.NilError(t, configfile.GetCredentialsStore(cred.ServerAddress).Store(cred))
			}
			loginErr := runLogin(context.Background(), cli, tc.inputLoginOption)
			if tc.expectedErr != "" {
				assert.Error(t, loginErr, tc.expectedErr)
				return
			}
			assert.NilError(t, loginErr)
			savedCred, credStoreErr := configfile.GetCredentialsStore(tc.inputStoredCred.ServerAddress).Get(tc.inputStoredCred.ServerAddress)
			assert.Check(t, credStoreErr)
			assert.DeepEqual(t, tc.expectedSavedCred, savedCred)
		})
	}
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
