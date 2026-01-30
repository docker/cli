package opts

import (
	"fmt"
)

// parseBoolValue returns the boolean value represented by the string. It returns
// true if no value is set.
//
// It is similar to [strconv.ParseBool], but only accepts 1, true, 0, false.
// Any other value returns an error.
func parseBoolValue(key string, val string, hasValue bool) (bool, error) {
	if !hasValue {
		return true, nil
	}
	switch val {
	case "1", "true":
		return true, nil
	case "0", "false":
		return false, nil
	default:
		return false, fmt.Errorf(`invalid value for '%s': invalid boolean value (%q): must be one of "true", "1", "false", or "0" (default "true")`, key, val)
	}
}
