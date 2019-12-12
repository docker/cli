package model

import "time"

// A CheckResponse is the internal content of the PublicCheckResponse signed
// json blob.
type CheckResponse struct {
	Expiration      time.Time `json:"expiration"`
	Token           string    `json:"token"`
	MaxEngines      int       `json:"maxEngines"`
	ScanningEnabled bool      `json:"scanningEnabled"`
	Type            string    `json:"licenseType"`
	Tier            string    `json:"tier"`

	SubscriptionID    string            `json:"subscription_id,omitempty"`
	ProductID         string            `json:"product_id,omitempty"`
	RatePlanID        string            `json:"rate_plan_id,omitempty"`
	Version           int               `json:"version"`
	GraceDays         int               `json:"grace_days,omitempty"`
	Metadata          *Metadata         `json:"metadata,omitempty"`
	PricingComponents PricingComponents `json:"pricing_components,omitempty"`
}

// Metadata holds non-essential license information, that is, anything that is not required by clients to ensure
// the license is valid
type Metadata struct {
	Username string `json:"username,omitempty"`
	Company  string `json:"company,omitempty"`
}

// IssuedLicense represents an issued license
type IssuedLicense struct {
	KeyID         string `json:"key_id"`
	PrivateKey    string `json:"private_key"`
	Authorization string `json:"authorization"`
}

// Valid returns true if the License is syntactically valid, false otherwise
func (l *IssuedLicense) Valid() (bool, string) {
	if l.KeyID == "" {
		return false, "empty key_id"
	}

	if l.PrivateKey == "" {
		return false, "empty private_key"
	}

	if l.Authorization == "" {
		return false, "empty authorization"
	}

	return true, ""
}
