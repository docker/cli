package jwt

import (
	"context"
)

type key int

var jwtContextKey key

// FromContext returns the token value stored in ctx, if any.
func FromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(jwtContextKey).(string)
	return token, ok
}

// NewContext returns a new Context that carries value token.
func NewContext(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, jwtContextKey, token)
}
