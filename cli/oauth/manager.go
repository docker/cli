package oauth

import (
	"context"
	"io"
)

// TokenResult is a result from the auth manager.
type TokenResult struct {
	AccessToken  string
	RefreshToken string
	RequireAuth  bool
	Tenant       string
	Claims       Claims
}

type Manager interface {
	LoginDevice(ctx context.Context, w io.Writer) (res TokenResult, err error)
	Logout() error
	RefreshToken() (res TokenResult, err error)
}
