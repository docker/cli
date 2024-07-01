package util

import (
	"errors"
	"os/exec"
	"runtime"
	"time"

	"github.com/docker/cli/cli/oauth"
	"github.com/go-jose/go-jose/v3/jwt"
)

// GetClaims returns claims from an access token without verification.
func GetClaims(accessToken string) (claims oauth.Claims, err error) {
	token, err := oauth.ParseSigned(accessToken)
	if err != nil {
		return
	}

	err = token.UnsafeClaimsWithoutVerification(&claims)

	return
}

// IsExpired returns whether the claims are expired or not.
func IsExpired(claims oauth.Claims) bool {
	err := claims.Validate(jwt.Expected{
		Time: time.Now().UTC(),
	})

	return errors.Is(err, jwt.ErrExpired)
}

// OpenBrowser opens the specified URL in a browser based on OS.
func OpenBrowser(url string) (err error) {
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = errors.New("unsupported platform")
	}

	return
}
