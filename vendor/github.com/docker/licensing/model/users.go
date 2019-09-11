package model

import (
	"fmt"
	"time"

	"github.com/docker/licensing/lib/errors"
)

// User details a Docker user
type User struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	DateJoined time.Time `json:"date_joined"`

	// The user type. Is either 'User' or 'Organization'
	Type string `json:"type"`

	FullName    string `json:"full_name,omitempty"`
	Location    string `json:"location,omitempty"`
	Company     string `json:"company,omitempty"`
	ProfileURL  string `json:"profile_url,omitempty"`
	GravatarURL string `json:"gravatar_url,omitempty"`
}

// Org details a Docker organization
type Org struct {
	ID         string    `json:"id"`
	Orgname    string    `json:"orgname"`
	DateJoined time.Time `json:"date_joined"`

	Type string `json:"type"`

	FullName    string `json:"full_name,omitempty"`
	Location    string `json:"location,omitempty"`
	Company     string `json:"company,omitempty"`
	ProfileURL  string `json:"profile_url,omitempty"`
	GravatarURL string `json:"gravatar_url,omitempty"`
}

// PaginationParams is used for specifying pagination in requests to accounts
type PaginationParams struct {
	PageSize int
	Page     int
}

// PaginatedMeta describes fields contained in a paginated response body
type PaginatedMeta struct {
	Count    int     `json:"count"`
	PageSize int     `json:"page_size,omitempty"`
	Next     *string `json:"next,omitempty"`
	Previous *string `json:"previous,omitempty"`
}

// LoginResult holds the response of the login endpoint
type LoginResult struct {
	// JWT associated with the authenticated user
	Token string `json:"token"`
}

// LoginRequest holds a hub user's username and password
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginError wraps both the http error and raw hub login error
type LoginError struct {
	*errors.HTTPError
	// Raw is the raw error from Accounts service
	Raw *RawLoginError
}

var _ error = (*LoginError)(nil)

func (e *LoginError) Error() string {
	msg := e.HTTPError.Error()
	if e.Raw != nil {
		msg = fmt.Sprintf("%s (raw: %+v)", msg, e.Raw)
	}

	return msg
}

// RawLoginError is the raw format of errors returned from the Accounts service.
type RawLoginError struct {
	Detail string `json:"detail,omitempty"`
	// These fields wil be populated if it's a validation error
	Username []string `json:"username,omitempty"`
	Password []string `json:"password,omitempty"`
}
