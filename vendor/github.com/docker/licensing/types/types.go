package types

// State represents a given subscription's current status
type State string

const (
	// Active means a subscription is currently in a working, live state
	Active State = "active"
	// Expired means a subscription's end date is in the past
	Expired State = "expired"
	// Cancelled means the subscription has been cancelled
	Cancelled State = "cancelled"
	// Preparing means that the subscription's payment (if any) is being still processed
	Preparing State = "preparing"
	// Failed means that there was a problem creating the subscription
	Failed State = "failed"
)
