package loader

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestV2Supported(t *testing.T) {
	actual, err := loadYAML(`
version: "2.0"
services:
  foo:
    image: busybox`)

	require.NoError(t, err)
	require.Len(t, actual.Services, 1)
}