package identity

import (
	"context"
	"fmt"
)

// DockerIdentity identifies a Docker user.
type DockerIdentity struct {
	DockerID string
	Username string
	FullName string
	Email    string
	Scopes   []string
}

func (di DockerIdentity) String() string {
	return fmt.Sprintf("{docker_id=%v, username=%v, email=%v, scopes=%v}",
		di.DockerID, di.Username, di.Email, di.Scopes)
}

// HasScope returns true if the exact input scope is present in the scopes list.
func (di DockerIdentity) HasScope(scope string) bool {
	for i := range di.Scopes {
		if di.Scopes[i] == scope {
			return true
		}
	}
	return false
}

type keyType int

var identityContextKey keyType

// FromContext returns the DockerIdentity value stored in ctx, if any.
func FromContext(ctx context.Context) (*DockerIdentity, bool) {
	identity, ok := ctx.Value(identityContextKey).(*DockerIdentity)
	return identity, ok
}

// NewContext returns a new Context that carries value identity.
func NewContext(ctx context.Context, identity *DockerIdentity) context.Context {
	return context.WithValue(ctx, identityContextKey, identity)
}
