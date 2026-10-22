// Copyright 2026 Docker, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package dockerhub reads Docker Hub access tokens and account profiles from
// the secrets engine. Obtain a [ClientAuth] from the client's HubAuth method,
// or from any [secrets.Resolver] via [New].
package dockerhub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/docker/secrets-engine/client/realms"
	"github.com/docker/secrets-engine/x/secrets"
)

var (
	// ErrNoSession means no Docker Hub credential is stored for the account.
	ErrNoSession = errors.New("user is not authenticated for this application")
	// ErrNoDefaultProfile means no account is set as the default. It wraps
	// [ErrNoSession].
	ErrNoDefaultProfile = fmt.Errorf("no default account profile set: %w", ErrNoSession)
	// ErrMissingAccessToken means the stored credential has no access token.
	// It wraps [ErrNoSession].
	ErrMissingAccessToken error = &noSessionError{"no access token in payload"}
	// ErrInvalidAccessToken means the stored access token is not a JWT. It
	// wraps [ErrNoSession].
	ErrInvalidAccessToken error = &noSessionError{"access token is not a JWT"}
)

type noSessionError struct{ msg string }

func (e *noSessionError) Error() string { return e.msg }

func (*noSessionError) Unwrap() error { return ErrNoSession }

var (
	defaultProfileKey = secrets.MustParsePattern("default")
	singleEntryKey    = secrets.MustParsePattern("*")
)

// UserSession is a stored Docker Hub credential.
type UserSession struct {
	// AccessToken is a Docker Hub issued JWT.
	AccessToken string `json:"access_token"`
	// Claims are zero when the payload carries none.
	Claims Claims `json:"claims"`
}

// Expired reports whether the access token has expired. It is
// ExpiresWithin(0).
func (s UserSession) Expired() bool {
	return s.ExpiresWithin(0)
}

// ExpiresWithin reports whether the access token will expire within the next given duration.
// Sessions without [Claims.ExpiresAt] report as expired by default.
func (s UserSession) ExpiresWithin(leeway time.Duration) bool {
	exp := s.Claims.ExpiresAt
	if exp == nil || exp.IsZero() {
		return true
	}
	return !time.Now().Add(leeway).Before(exp.Time)
}

// Claims are the claims of a Docker Hub access token.
type Claims struct {
	Issuer    string       `json:"iss,omitempty"`
	Subject   string       `json:"sub,omitempty"`
	Audience  Audience     `json:"aud,omitempty"`
	ExpiresAt *NumericDate `json:"exp,omitempty"`
	NotBefore *NumericDate `json:"nbf,omitempty"`
	IssuedAt  *NumericDate `json:"iat,omitempty"`
	ID        string       `json:"jti,omitempty"`

	// Scope is a space-delimited list of granted scopes.
	Scope string `json:"scope,omitempty"`
	// AppName is the Docker client application the token was issued to.
	AppName string `json:"app_name"`
	UUID    string `json:"uuid"`
	// Source is formatted as `docker_{type}|{id}`.
	Source     string `json:"source"`
	SessionID  string `json:"session_id"`
	ClientID   string `json:"client_id,omitempty"`
	ClientName string `json:"client_name,omitempty"`
	Email      string `json:"email"`
	Username   string `json:"username"`
}

// NumericDate is an RFC 7519 numeric date: UNIX epoch seconds. It marshals
// truncated to whole seconds.
type NumericDate struct {
	time.Time
}

func (d NumericDate) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatInt(d.Unix(), 10)), nil
}

func (d *NumericDate) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var epoch float64
	if err := json.Unmarshal(data, &epoch); err != nil {
		return fmt.Errorf("parse numeric date: %w", err)
	}
	if math.Abs(epoch) > 1e15 {
		return fmt.Errorf("parse numeric date: %v is out of range", epoch)
	}
	seconds, fraction := math.Modf(epoch)
	d.Time = time.Unix(int64(seconds), int64(fraction*float64(time.Second)))
	return nil
}

// Audience is the "aud" claim: a single string or an array of strings.
type Audience []string

func (a *Audience) UnmarshalJSON(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("parse audience: %w", err)
	}
	switch v := value.(type) {
	case nil:
	case string:
		*a = Audience{v}
	case []any:
		audience := make(Audience, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return fmt.Errorf("parse audience: unexpected element type %T", item)
			}
			audience = append(audience, s)
		}
		*a = audience
	default:
		return fmt.Errorf("parse audience: unexpected type %T", value)
	}
	return nil
}

// Profile describes a signed-in Docker Hub account.
type Profile struct {
	// UserID is the secret ID where the account's credential is stored.
	UserID string `json:"user_id"`
	// OriginalSignInApp is the Docker client application the user signed in from.
	OriginalSignInApp string    `json:"original_sign_in_app"`
	Username          string    `json:"username"`
	Email             string    `json:"email"`
	SignInDate        time.Time `json:"sign_in_date"`
}

// parseUserSession decodes a docker/auth/hub/** envelope into a [UserSession].
func parseUserSession(envelope secrets.Envelope) (UserSession, error) {
	var session UserSession
	if err := json.Unmarshal(envelope.Value, &session); err != nil {
		return UserSession{}, fmt.Errorf("decode user session: %w", err)
	}
	if session.AccessToken == "" {
		return UserSession{}, fmt.Errorf("decode user session: %w", ErrMissingAccessToken)
	}
	if err := checkJWT(session.AccessToken); err != nil {
		return UserSession{}, fmt.Errorf("decode user session: %w: %w", ErrInvalidAccessToken, err)
	}
	return session, nil
}

func checkJWT(token string) error {
	segments := strings.Split(token, ".")
	if len(segments) != 3 {
		return fmt.Errorf("expected 3 dot-separated segments, got %d", len(segments))
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := decodeSegment(segments[0], &header); err != nil {
		return fmt.Errorf("header: %w", err)
	}
	if header.Alg == "" {
		return errors.New("header: missing alg")
	}
	var payload map[string]json.RawMessage
	if err := decodeSegment(segments[1], &payload); err != nil {
		return fmt.Errorf("payload: %w", err)
	}
	if payload == nil {
		return errors.New("payload: not a JSON object")
	}
	if segments[2] == "" {
		return errors.New("missing signature")
	}
	if _, err := base64.RawURLEncoding.DecodeString(segments[2]); err != nil {
		return fmt.Errorf("signature: %w", err)
	}
	return nil
}

func decodeSegment(segment string, v any) error {
	data, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// parseProfile decodes a docker/auth/metadata/hub/** envelope into a [Profile].
func parseProfile(envelope secrets.Envelope) (Profile, error) {
	var profile Profile
	if err := json.Unmarshal(envelope.Value, &profile); err != nil {
		return Profile{}, fmt.Errorf("decode profile metadata: %w", err)
	}
	if profile.UserID == "" {
		return Profile{}, errors.New("decode profile metadata: no user ID in payload")
	}
	return profile, nil
}

// ClientAuth reads Docker Hub authentication state from the secrets engine.
type ClientAuth interface {
	// ListProfiles returns the profiles of all signed-in accounts.
	ListProfiles(ctx context.Context) ([]Profile, error)
	// GetDefaultProfile returns the default account's profile, or
	// [ErrNoDefaultProfile] when no default is set.
	GetDefaultProfile(ctx context.Context) (Profile, error)
	// GetDefaultSession returns the default account's session:
	// [ErrNoDefaultProfile] when no default is set, [ErrNoSession] when its
	// credential is missing.
	GetDefaultSession(ctx context.Context) (UserSession, error)
	// GetSession returns the session for username, or [ErrNoSession].
	GetSession(ctx context.Context, username string) (UserSession, error)
}

// Option configures a [ClientAuth].
type Option func(*config)

type config struct {
	accounts     secrets.Pattern
	profiles     secrets.Pattern
	defaultEntry secrets.Pattern
	accountEntry secrets.Pattern
}

// Staging switches the lookup to the Docker Hub staging realms.
func Staging() Option {
	return func(c *config) {
		c.accounts = realms.DockerHubStagingAuthentication
		c.profiles = realms.DockerHubStagingAuthenticationMetadata
	}
}

func newConfig(opts []Option) config {
	cfg := config{
		accounts: realms.DockerHubAuthentication,
		profiles: realms.DockerHubAuthenticationMetadata,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.defaultEntry = mustExpand(cfg.profiles, defaultProfileKey)
	cfg.accountEntry = mustExpand(cfg.accounts, singleEntryKey)
	return cfg
}

func mustExpand(realm, key secrets.Pattern) secrets.Pattern {
	expanded, err := realm.ExpandPattern(key)
	if err != nil {
		panic(fmt.Sprintf("dockerhub: expand %s in %s: %v", key, realm, err))
	}
	return expanded
}

var _ ClientAuth = clientAuth{}

type clientAuth struct {
	engine secrets.Resolver
	cfg    config
}

// New returns a [ClientAuth] backed by engine. It panics on a nil engine.
func New(engine secrets.Resolver, opts ...Option) ClientAuth {
	if engine == nil {
		panic("dockerhub: secrets engine client is required")
	}
	return clientAuth{engine: engine, cfg: newConfig(opts)}
}

func (c clientAuth) ListProfiles(ctx context.Context) ([]Profile, error) {
	envelopes, err := c.engine.GetSecrets(ctx, c.cfg.profiles)
	if errors.Is(err, secrets.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list account profiles: %w", err)
	}
	var profiles []Profile
	var errs []error
	seen := make(map[string]bool, len(envelopes))
	for _, envelope := range envelopes {
		// The default entry duplicates an account's profile.
		if envelope.ID != nil && c.cfg.defaultEntry.Match(envelope.ID) {
			continue
		}
		profile, err := parseProfile(envelope)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if seen[profile.UserID] {
			continue
		}
		seen[profile.UserID] = true
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 && len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return profiles, nil
}

func (c clientAuth) GetDefaultProfile(ctx context.Context) (Profile, error) {
	envelopes, err := c.engine.GetSecrets(ctx, c.cfg.defaultEntry)
	if errors.Is(err, secrets.ErrNotFound) {
		return Profile{}, ErrNoDefaultProfile
	}
	if err != nil {
		return Profile{}, fmt.Errorf("retrieve default account metadata: %w", err)
	}
	return parseFirst(envelopes, parseProfile, ErrNoDefaultProfile)
}

func (c clientAuth) GetDefaultSession(ctx context.Context) (UserSession, error) {
	profile, err := c.GetDefaultProfile(ctx)
	if err != nil {
		return UserSession{}, err
	}
	// Require a wildcard-free ID naming one account entry in the accounts
	// realm, so a tampered profile cannot address an arbitrary secret.
	id, err := secrets.ParseID(profile.UserID)
	if err != nil {
		return UserSession{}, fmt.Errorf("default profile user id: %w", err)
	}
	if !c.cfg.accountEntry.Match(id) {
		return UserSession{}, fmt.Errorf("default profile user id %q is not an account entry in the %s realm", profile.UserID, c.cfg.accounts)
	}
	return c.getSession(ctx, exactPattern(id))
}

func (c clientAuth) GetSession(ctx context.Context, username string) (UserSession, error) {
	if strings.Contains(username, "/") {
		return UserSession{}, fmt.Errorf("invalid username %q: must not contain '/'", username)
	}
	user, err := secrets.ParseID(username)
	if err != nil {
		return UserSession{}, fmt.Errorf("invalid username %q: %w", username, err)
	}
	id, err := c.cfg.accounts.ExpandID(user)
	if err != nil {
		return UserSession{}, err
	}
	return c.getSession(ctx, exactPattern(id))
}

func (c clientAuth) getSession(ctx context.Context, pattern secrets.Pattern) (UserSession, error) {
	envelopes, err := c.engine.GetSecrets(ctx, pattern)
	if errors.Is(err, secrets.ErrNotFound) {
		return UserSession{}, ErrNoSession
	}
	if err != nil {
		return UserSession{}, fmt.Errorf("retrieve user access token: %w", err)
	}
	return parseFirst(envelopes, parseUserSession, ErrNoSession)
}

func exactPattern(id secrets.ID) secrets.Pattern {
	return secrets.MustParsePattern(id.String())
}

// parseFirst returns the first envelope that parses. It returns notFound when
// there are no envelopes, and the joined decode errors when none parses.
func parseFirst[T any](envelopes []secrets.Envelope, parse func(secrets.Envelope) (T, error), notFound error) (T, error) {
	var zero T
	var errs []error
	for _, envelope := range envelopes {
		v, err := parse(envelope)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return v, nil
	}
	if len(errs) == 0 {
		return zero, notFound
	}
	return zero, errors.Join(errs...)
}
