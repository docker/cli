package registry

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/internal/registry"
	"github.com/docker/cli/internal/tui"
	registrytypes "github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
	"github.com/morikuni/aec"
	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
)

// tryRegistryDeviceLogin attempts a browser login when the registry advertises one.
// The boolean is false when the caller should fall back to a password prompt.
func tryRegistryDeviceLogin(ctx context.Context, dockerCLI command.Cli, serverAddress string) (string, error, bool) {
	svc, err := registry.NewService(registry.ServiceOptions{})
	if err != nil {
		logrus.WithError(err).Debug("device login unavailable")
		return "", nil, false
	}
	endpoints, err := svc.Endpoints(ctx, serverAddress)
	if err != nil || len(endpoints) == 0 {
		return "", nil, false
	}

	var offer registry.DeviceLogin
	var found bool
	for _, endpoint := range endpoints {
		offer, err = registry.DiscoverDeviceLogin(ctx, endpoint, command.UserAgent())
		if err == nil {
			found = true
			break
		}
		if errors.Is(err, registry.ErrDeviceLoginStartFail) {
			logrus.WithError(err).Debug("device login challenge was not usable")
			_, _ = fmt.Fprint(dockerCLI.Err(), "Failed to start web-based login - falling back to command line login...\n\n")
			return "", nil, false
		}
		logrus.WithError(err).Debug("registry did not offer device login")
	}
	if !found {
		return "", nil, false
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	code, err := registry.RequestDeviceCode(ctx, httpClient, offer.DeviceEndpoint)
	if err != nil {
		logrus.WithError(err).Debug("failed to start registry device login")
		_, _ = fmt.Fprint(dockerCLI.Err(), "Failed to start web-based login - falling back to command line login...\n\n")
		return "", nil, false
	}

	cred, err := waitForDeviceCredential(ctx, dockerCLI, httpClient, offer.TokenEndpoint, code)
	if err != nil {
		return "", err, true
	}

	response, err := loginWithRegistry(ctx, dockerCLI.Client(), client.RegistryLoginOptions{
		Username:      cred.Username,
		Password:      cred.Secret,
		ServerAddress: serverAddress,
		IdentityToken: cred.IdentityToken,
	})
	if err != nil {
		return "", err, true
	}
	stored := registrytypes.AuthConfig{
		Username:      cred.Username,
		Password:      cred.Secret,
		ServerAddress: serverAddress,
		IdentityToken: cred.IdentityToken,
	}
	if response.Auth.IdentityToken != "" {
		stored.Password = ""
		stored.IdentityToken = response.Auth.IdentityToken
	}
	if err := storeCredentials(dockerCLI.ConfigFile(), stored); err != nil {
		return "", err, true
	}
	return response.Auth.Status, nil, true
}

func waitForDeviceCredential(ctx context.Context, dockerCLI command.Cli, httpClient *http.Client, tokenEndpoint string, code registry.DeviceCode) (registry.DeviceCredential, error) {
	ctx, cancel := context.WithTimeout(ctx, code.ExpiresIn)
	defer cancel()

	out := tui.NewOutput(dockerCLI.Err())
	out.PrintNote("Using web-based login. To sign in with credentials on the command line, use 'docker login -u <username>'\n")
	_, _ = fmt.Fprintln(dockerCLI.Err(), "Your one-time device confirmation code is:", aec.Bold.Apply(code.UserCode))
	_, _ = fmt.Fprintln(dockerCLI.Err(), "Open this URL to authenticate:", aec.Underline.Apply(code.VerificationURI))
	_, _ = fmt.Fprintln(dockerCLI.Err(), "\n"+aec.Bold.Apply("Press ENTER")+" to open the URL in your browser, or "+aec.Bold.Apply("Ctrl+C")+" to cancel.")

	openURL := code.VerificationURI
	if code.VerificationURIComplete != "" {
		openURL = code.VerificationURIComplete
	}

	credCh := make(chan registry.DeviceCredential, 1)
	errCh := make(chan error, 1)
	go func() {
		cred, err := registry.PollDeviceToken(ctx, httpClient, tokenEndpoint, code.DeviceCode, code.Interval, nil)
		if err != nil {
			errCh <- err
			return
		}
		credCh <- cred
	}()
	go func() {
		reader := bufio.NewReader(dockerCLI.In())
		if _, err := reader.ReadString('\n'); err != nil || ctx.Err() != nil {
			return
		}
		if err := browser.OpenURL(openURL); err != nil {
			_, _ = fmt.Fprintln(dockerCLI.Err(), "Open the URL above manually to continue.")
			return
		}
		_, _ = fmt.Fprintln(dockerCLI.Err(), "\nOpening browser…")
	}()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return registry.DeviceCredential{}, errors.New("timed out waiting for the browser login")
		}
		return registry.DeviceCredential{}, errors.New("login canceled")
	case err := <-errCh:
		return registry.DeviceCredential{}, fmt.Errorf("failed waiting for authentication: %w", err)
	case cred := <-credCh:
		return cred, nil
	}
}
