package model

import (
	"github.com/docker/licensing/lib/go-validation"
)

// Profile represents an Account profile
type Profile struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`

	Addresses    []*Address `json:"addresses,omitempty"`
	CompanyName  string     `json:"company_name,omitempty"`
	PhonePrimary string     `json:"phone_primary,omitempty"`
	JobFunction  string     `json:"job_function,omitempty"`
	VatID        string     `json:"vat_id,omitempty"`
}

// Address represents a Profile address
type Address struct {
	AddressLine1   string `json:"address_line_1,omitempty"`
	AddressLine2   string `json:"address_line_2,omitempty"`
	AddressLine3   string `json:"address_line_3,omitempty"`
	City           string `json:"city,omitempty"`
	Province       string `json:"province,omitempty"`
	Country        string `json:"country,omitempty"`
	Postcode       string `json:"post_code,omitempty"`
	PrimaryAddress bool   `json:"primary_address,omitempty"`
}

// Account represents a billing profile
type Account struct {
	DockerID string `json:"docker_id"`

	Profile Profile `json:"profile"`
}

// AccountCreationRequest represents an Account creation request
type AccountCreationRequest struct {
	Profile Profile `json:"profile"`
}

// Validate returns true if the account request is valid, false otherwise.
// If invalid, one or more validation Errors will be returned.
func (a *AccountCreationRequest) Validate() (bool, validation.Errors) {
	profile := a.Profile

	var errs validation.Errors

	if validation.IsEmpty(profile.Email) {
		errs = append(errs, validation.InvalidEmpty("email"))
	}

	if !validation.IsEmpty(profile.Email) && !validation.IsEmail(profile.Email) {
		errs = append(errs, validation.InvalidEmail("email", profile.Email))
	}

	valid := len(errs) == 0
	return valid, errs
}
