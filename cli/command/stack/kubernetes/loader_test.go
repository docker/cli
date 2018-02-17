package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlaceholders(t *testing.T) {
	env := map[string]string{
		"TAG": "_latest_",
		"K1":  "V1",
		"K2":  "V2",
	}

	prefix := "version: '3'\nvolumes:\n  data:\n    external:\n      name: "
	var tests = []struct {
		input          string
		expectedOutput string
	}{
		{prefix + "BEFORE${TAG}AFTER", prefix + "BEFORE_latest_AFTER"},
		{prefix + "BEFORE${K1}${K2}AFTER", prefix + "BEFOREV1V2AFTER"},
		{prefix + "BEFORE$TAG AFTER", prefix + "BEFORE_latest_ AFTER"},
		{prefix + "BEFORE$$TAG AFTER", prefix + "BEFORE$TAG AFTER"},
		{prefix + "BEFORE $UNKNOWN AFTER", prefix + "BEFORE  AFTER"},
	}

	for _, test := range tests {
		output, _, err := load("stack", []byte(test.input), ".", env)
		require.NoError(t, err)
		assert.Equal(t, test.expectedOutput, output.Spec.ComposeFile)
	}
}
