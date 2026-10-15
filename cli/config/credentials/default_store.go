package credentials

import "os/exec"

// DetectDefaultStore returns the credentials store to use if no user-defined
// custom helper is passed.
//
// Some platforms define a preferred helper, in which case it attempts to look
// up the helper binary before falling back to the platform's default.
//
// If no user-defined helper is passed, and no helper is found, it returns an
// empty string, which means credentials are stored unencrypted in the CLI's
// config-file without the use of a credentials store.
func DetectDefaultStore(customStore string) string {
	if customStore != "" {
		// use user-defined
		return customStore
	}
	if preferred := findPreferredHelper(); preferred != "" {
		return preferred
	}
	if defaultHelper == "" {
		return ""
	}
	if _, err := exec.LookPath(remoteCredentialsPrefix + defaultHelper); err != nil {
		return ""
	}
	return defaultHelper
}

// overridePreferred is used to override the preferred helper in tests.
var overridePreferred string

// findPreferredHelper detects whether the preferred credentials-store and
// its helper binaries are installed. It returns the name of the preferred
// store if found, otherwise returns an empty string to fall back to resolving
// the default helper.
//
// Note that the logic below is currently specific to detection needed for the
// "pass" credentials-helper on Linux (which is the only platform with a preferred
// helper). It is put in a non-platform specific file to allow running tests
// on other platforms.
func findPreferredHelper() string {
	preferred := preferredHelper
	if overridePreferred != "" {
		preferred = overridePreferred
	}
	if preferred == "" {
		return ""
	}

	// Note that the logic below is specific to detection needed for the
	// "pass" credentials-helper on Linux (which is the only platform with
	// a "preferred" and "default" helper. This logic may change if a similar
	// order of preference is needed on other platforms.

	// If we don't have the preferred helper installed, there's no need
	// to check if its dependencies are installed, instead, try to
	// use the default credentials-helper for this platform (if installed).
	if _, err := exec.LookPath(remoteCredentialsPrefix + preferred); err != nil {
		return ""
	}

	// Detect if the helper binary is present as well. This is needed for
	// the "pass" credentials helper, which uses this binary.
	if _, err := exec.LookPath(preferred); err != nil {
		return ""
	}

	return preferred
}
