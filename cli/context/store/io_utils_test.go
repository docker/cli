package store

import (
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"gotest.tools/v3/assert"
)

func TestLimitReaderReadAll(t *testing.T) {
	r := strings.NewReader("Reader")

	_, err := io.ReadAll(r)
	assert.NilError(t, err)

	r = strings.NewReader("Test")
	_, err = io.ReadAll(&limitedReader{R: r, N: 4})
	assert.NilError(t, err)

	r = strings.NewReader("Test")
	_, err = io.ReadAll(&limitedReader{R: r, N: 2})
	assert.Error(t, err, "read exceeds the defined limit")

	// exceeding the limit must error even when a read lands exactly on
	// the limit before the remaining data is reached
	r = strings.NewReader("Test")
	_, err = io.ReadAll(&limitedReader{R: iotest.OneByteReader(r), N: 2})
	assert.Error(t, err, "read exceeds the defined limit")

	// a reader is allowed to return data together with io.EOF; the limit
	// must still be reported, otherwise io.ReadAll treats the truncated
	// content as a successful read
	r = strings.NewReader("Tes")
	_, err = io.ReadAll(&limitedReader{R: iotest.DataErrReader(r), N: 2})
	assert.Error(t, err, "read exceeds the defined limit")
}
