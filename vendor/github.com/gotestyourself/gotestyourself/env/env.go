/*Package env provides functions to test code that read environment variables
 */
package env

import (
	"os"
	"strings"

	"github.com/stretchr/testify/require"
)

// Patch changes the value of an environment variable, and returns a
// function which will reset the the value of that variable back to the
// previous state.
func Patch(t require.TestingT, key, value string) func() {
	oldValue, ok := os.LookupEnv(key)
	require.NoError(t, os.Setenv(key, value))
	return func() {
		if !ok {
			require.NoError(t, os.Unsetenv(key))
			return
		}
		require.NoError(t, os.Setenv(key, oldValue))
	}
}

// PatchAll sets the environment to env, and returns a function which will
// reset the environment back to the previous state.
func PatchAll(t require.TestingT, env map[string]string) func() {
	oldEnv := os.Environ()
	os.Clearenv()

	for key, value := range env {
		require.NoError(t, os.Setenv(key, value))
	}
	return func() {
		os.Clearenv()
		for key, oldVal := range ToMap(oldEnv) {
			require.NoError(t, os.Setenv(key, oldVal))
		}
	}
}

// ToMap takes a list of strings in the format returned by os.Environ() and
// returns a mapping of keys to values.
func ToMap(env []string) map[string]string {
	result := map[string]string{}
	for _, raw := range env {
		parts := strings.SplitN(raw, "=", 2)
		switch len(parts) {
		case 1:
			result[raw] = ""
		case 2:
			result[parts[0]] = parts[1]
		}
	}
	return result
}
