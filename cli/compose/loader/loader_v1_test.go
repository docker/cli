package loader

import (
	"testing"
	"github.com/stretchr/testify/require"
)


func TestUnsupportedVersion(t *testing.T) {
	_, err := loadYAML(`
version: "0.1"
services:
  foo:
    image: busybox
`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "version")

	_, err = loadYAML(`
version: "0.1"
services:
  foo:
    image: busybox
`)
	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}


func TestV1Supported(t *testing.T) {
	actual, err := loadYAML(`
foo:
  image: busybox
`)
	require.NoError(t, err)
	require.Len(t, actual.Services, 1)
}
