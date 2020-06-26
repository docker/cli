package opts

import (
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/env"
	"gotest.tools/v3/fs"
)

func TestReadKVEnvStrings(t *testing.T) {
	emptyEnvFile := fs.NewFile(t, t.Name())
	defer emptyEnvFile.Remove()

	// EMPTY_VAR should be set, but with an empty string as value
	// FROM_ENV value should be substituted with FROM_ENV in the current environment
	// NO_SUCH_ENV does not exist in the current environment, so should be omitted
	envFile1 := fs.NewFile(t, t.Name(), fs.WithContent(`Z1=z
EMPTY_VAR=
FROM_ENV
NO_SUCH_ENV
`))
	defer envFile1.Remove()
	envFile2 := fs.NewFile(t, t.Name(), fs.WithContent("Z2=z\nA2=a"))
	defer envFile2.Remove()
	defer env.Patch(t, "FROM_ENV", "from-env")()

	tests := []struct {
		name      string
		files     []string
		overrides []string
		expected  []string
	}{
		{
			name: "empty",
		},
		{
			name:  "empty file",
			files: []string{emptyEnvFile.Path()},
		},
		{
			name:     "single file",
			files:    []string{envFile1.Path()},
			expected: []string{"Z1=z", "EMPTY_VAR=", "FROM_ENV=from-env"},
		},
		{
			name:     "two files",
			files:    []string{envFile1.Path(), envFile2.Path()},
			expected: []string{"Z1=z", "EMPTY_VAR=", "FROM_ENV=from-env", "Z2=z", "A2=a"},
		},
		{
			name:      "single file and override",
			files:     []string{envFile1.Path()},
			overrides: []string{"Z1=override", "EXTRA=extra"},
			expected:  []string{"Z1=z", "EMPTY_VAR=", "FROM_ENV=from-env", "Z1=override", "EXTRA=extra"},
		},
		{
			name:      "overrides only",
			overrides: []string{"Z1=z", "EMPTY_VAR="},
			expected:  []string{"Z1=z", "EMPTY_VAR="},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			envs, err := ReadKVEnvStrings(tc.files, tc.overrides)
			assert.NilError(t, err)
			assert.DeepEqual(t, tc.expected, envs)
		})
	}
}
