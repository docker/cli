package jwt

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/docker/licensing/lib/go-auth/identity"
	"github.com/google/uuid"
)

const (
	// x509 cert chain header field
	// https://tools.ietf.org/html/draft-ietf-jose-json-web-key-41#section-4.7
	x5c = "x5c"

	// non standard username claim
	username = "username"

	// non standard email claim
	email = "email"

	// subject claim
	// https://tools.ietf.org/html/rfc7519#section-4.1.2
	sub = "sub"

	// jwt id claim
	// https://tools.ietf.org/html/rfc7519#section-4.1.7
	jti = "jti"

	// issued at claim
	// https://tools.ietf.org/html/rfc7519#section-4.1.6
	iat = "iat"

	// expiration time claim
	// https://tools.ietf.org/html/rfc7519#section-4.1.4
	exp = "exp"

	// Legacy claims the gateways are still using to validate a JWT.
	sessionid = "session_id" // same as a `jti`
	userid    = "user_id"    // same as a `sub`

	// non standard scope claim
	scope = "scope"
)

// EncodeOptions holds JWT encoding options
type EncodeOptions struct {
	// The token expiration time, represented as a UNIX timestamp
	Expiration int64

	// TODO would be good to add a leeway option, but go-jwt
	// does not support this. see: https://github.com/dgrijalva/jwt-go/issues/131

	// The private key with which to sign the token
	SigningKey []byte

	// The x509 certificate associated with the signing key
	Certificate []byte

	// Identifier for the JWT. If this is empty, a random UUID will be generated.
	Jti string

	// Whether or not to include legacy claims that the gateways are still using to validate a JWT.
	IncludeLegacyClaims bool
}

// Encode creates a JWT string for the given identity.DockerIdentity.
func Encode(identity identity.DockerIdentity, options EncodeOptions) (string, error) {
	// Note: we only support a RS256 signing method right now. If we want to support
	// additional signing methods (for example, HS256), this could be specified as an
	// encoding option.
	token := jwt.New(jwt.SigningMethodRS256)

	block, _ := pem.Decode(options.Certificate)
	if block == nil {
		return "", fmt.Errorf("invalid key: failed to parse header")
	}

	encodedCert := base64.StdEncoding.EncodeToString(block.Bytes)
	x5cCerts := [1]string{encodedCert}

	token.Header[x5c] = x5cCerts

	// non standard fields
	// Note: this is a required field
	token.Claims[username] = identity.Username
	token.Claims[email] = identity.Email

	// standard JWT fields, consult the JWT spec for details
	token.Claims[sub] = identity.DockerID

	if len(identity.Scopes) > 0 {
		token.Claims[scope] = strings.Join(identity.Scopes, " ")
	}

	jtiStr := options.Jti
	if len(jtiStr) == 0 {
		jtiStr = "jti-" + uuid.New().String()
	}
	token.Claims[jti] = jtiStr

	token.Claims[iat] = time.Now().Unix()
	token.Claims[exp] = options.Expiration

	if options.IncludeLegacyClaims {
		token.Claims[sessionid] = jtiStr
		token.Claims[userid] = identity.DockerID
	}

	return token.SignedString(options.SigningKey)
}

// DecodeOptions holds JWT decoding options
type DecodeOptions struct {
	CertificateChain *x509.CertPool
}

// Decode decodes the given JWT string, returning the decoded identity.DockerIdentity
func Decode(tokenStr string, options DecodeOptions) (*identity.DockerIdentity, error) {
	rootCerts := options.CertificateChain
	token, err := jwt.Parse(tokenStr, keyFunc(rootCerts))

	if err == nil && token.Valid {
		username, ok := token.Claims[username].(string)
		if !ok {
			return nil, fmt.Errorf("%v claim not present", username)
		}
		dockerID, ok := token.Claims[sub].(string)
		if !ok {
			return nil, fmt.Errorf("%v claim not present", sub)
		}

		// email is optional
		email, _ := token.Claims[email].(string)

		var scopes []string
		if scopeClaim, ok := token.Claims[scope]; ok {
			sstr, ok := scopeClaim.(string)
			if !ok {
				return nil, fmt.Errorf("scope claim invalid")
			}
			scopes = strings.Split(sstr, " ")
		}

		identity := &identity.DockerIdentity{
			Username: username,
			DockerID: dockerID,
			Email:    email,
			Scopes:   scopes,
		}
		return identity, nil
	}

	// no error but an invalid token seems like a corner case, but just to be sure
	if err == nil && !token.Valid {
		return nil, fmt.Errorf("token was invalid")
	}

	if ve, ok := err.(*jwt.ValidationError); ok {
		return nil, &ValidationError{VError: ve}
	}
	return nil, fmt.Errorf("error decoding token: %s", err)
}

// IsExpired returns true if the token has expired, false otherwise
func IsExpired(tokenStr string, options DecodeOptions) (bool, error) {
	rootCerts := options.CertificateChain
	_, err := jwt.Parse(tokenStr, keyFunc(rootCerts))
	if err == nil {
		return false, nil
	}

	if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&(jwt.ValidationErrorExpired) != 0 {
			return true, nil
		}

		return false, err
	}

	return false, err
}

// keyFunc returns the jwt.KeyFunc with which to validate the token
func keyFunc(roots *x509.CertPool) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// x5c holds a base64 encoded DER encoded x509 certificate
		// associated with the private key used to sign the token.

		// For more information, see:
		// https://tools.ietf.org/html/draft-ietf-jose-json-web-key-41#page-9
		// https://tools.ietf.org/html/draft-ietf-jose-json-web-key-41#appendix-B
		x5c, ok := token.Header[x5c].([]interface{})
		if !ok {
			return nil, fmt.Errorf("x5c token header not present")
		}

		if len(x5c) == 0 {
			return nil, fmt.Errorf("x5c token header was empty")
		}

		x5cString, ok := x5c[0].(string)
		if !ok {
			return nil, fmt.Errorf("x5c token header was not a string")
		}

		decodedCert, err := base64.StdEncoding.DecodeString(x5cString)
		if err != nil {
			return nil, err
		}

		cert, err := validateCert(decodedCert, roots)
		if err != nil {
			return nil, err
		}

		key := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})

		return key, nil
	}
}

// validateCert validates the ASN.1 DER encoded cert using the given x509.CertPool root
// certificate chain. If valid, the parsed x509.Certificate is returned.
func validateCert(derData []byte, roots *x509.CertPool) (*x509.Certificate, error) {
	opts := x509.VerifyOptions{
		Roots: roots,
	}

	cert, err := x509.ParseCertificate(derData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: [%v]", err)
	}

	_, err = cert.Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to verify certificate: [%v]", err)
	}

	return cert, nil
}

// ValidationError interrogates the jwt.ValidationError, returning
// a more detailed error message.
type ValidationError struct {
	VError *jwt.ValidationError
}

func (e *ValidationError) Error() string {
	errs := e.VError.Errors

	if errs&jwt.ValidationErrorMalformed != 0 {
		return fmt.Sprintf("malformed token error: [%v]", e.VError)
	}

	if errs&jwt.ValidationErrorUnverifiable != 0 {
		return fmt.Sprintf("token signature error: [%v]", e.VError)
	}

	if errs&jwt.ValidationErrorSignatureInvalid != 0 {
		return fmt.Sprintf("token signature error: [%v]", e.VError)
	}

	if errs&jwt.ValidationErrorExpired != 0 {
		return fmt.Sprintf("token expiration error: [%v]", e.VError)
	}

	if errs&jwt.ValidationErrorNotValidYet != 0 {
		return fmt.Sprintf("token NBF validation error: [%v]", e.VError)
	}

	return fmt.Sprintf("token validation error: [%v]", e.VError)
}
