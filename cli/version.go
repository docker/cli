package cli

// Default build-time variable.
// These values are overriding via ldflags
var (
	Version   = "unknown-version-1"
	GitCommit = "unknown-commit"
	BuildTime = "unknown-buildtime"
)
