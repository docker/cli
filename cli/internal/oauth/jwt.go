package oauth

import (
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// Claims represents standard claims along with some custom ones.
type Claims struct {
	jwt.Claims

	// Domain is the domain claims for the token.
	Domain DomainClaims `json:"https://hub.docker.com"`

	// Scope is the scopes for the claims as a string that is space delimited.
	Scope string `json:"scope,omitempty"`
}

// DomainClaims represents a custom claim data set that doesn't change the spec
// payload. This is primarily introduced by Auth0 and is defined by a fully
// specified URL as it's key. e.g. "https://hub.docker.com"
type DomainClaims struct {
	// UUID is the user, machine client, or organization's UUID in our database.
	UUID string `json:"uuid"`

	// Email is the user's email address.
	Email string `json:"email"`

	// Username is the user's username.
	Username string `json:"username"`

	// Source is the source of the JWT. This should look like
	// `docker_{type}|{id}`.
	Source string `json:"source"`

	// SessionID is the unique ID of the token.
	SessionID string `json:"session_id"`

	// ClientID is the client_id that generated the token. This is filled if
	// M2M.
	ClientID string `json:"client_id,omitempty"`

	// ClientName is the name of the client that generated the token. This is
	// filled if M2M.
	ClientName string `json:"client_name,omitempty"`
}

// Source represents a source of a JWT.
type Source struct {
	// Type is the type of source. This could be "pat" etc.
	Type string `json:"type"`

	// ID is the identifier to the source type. If "pat" then this will be the
	// ID of the PAT.
	ID string `json:"id"`
}

// GetClaims returns claims from an access token without verification.
func GetClaims(accessToken string) (claims Claims, err error) {
	token, err := parseSigned(accessToken)
	if err != nil {
		return
	}

	err = token.UnsafeClaimsWithoutVerification(&claims)

	return
}

// allowedSignatureAlgorithms is a list of allowed signature algorithms for JWTs.
// We add all supported algorithms for Auth0, including with higher key lengths.
// See auth0 docs: https://auth0.com/docs/get-started/applications/signing-algorithms
var allowedSignatureAlgorithms = []jose.SignatureAlgorithm{
	jose.HS256,
	jose.HS384,
	jose.HS512,
	jose.RS256, // currently used for auth0
	jose.RS384,
	jose.RS512,
	jose.PS256,
	jose.PS384,
	jose.PS512,
}

// parseSigned parses a JWT and returns the signature object or error. This does
// not verify the validity of the JWT.
func parseSigned(token string) (*jwt.JSONWebToken, error) {
	return jwt.ParseSigned(token, allowedSignatureAlgorithms)
}
