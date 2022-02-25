package store

import (
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestLimitReaderReadAll(t *testing.T) {
	r := strings.NewReader("Reader")

	_, err := io.ReadAll(r)
	assert.NilError(t, err)

	r = strings.NewReader("Test")
	_, err = io.ReadAll(&LimitedReader{R: r, N: 4})
	assert.NilError(t, err)

	r = strings.NewReader("Test")
	_, err = io.ReadAll(&LimitedReader{R: r, N: 2})
	assert.Error(t, err, "read exceeds the defined limit")
}
