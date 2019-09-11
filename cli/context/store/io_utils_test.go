package store

import (
	"io/ioutil"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestLimitReaderReadAll(t *testing.T) {
	r := strings.NewReader("Reader")

	_, err := ioutil.ReadAll(r)
	assert.NilError(t, err)

	r = strings.NewReader("Test")
	_, err = ioutil.ReadAll(&LimitedReader{R: r, N: 4})
	assert.NilError(t, err)

	r = strings.NewReader("Test")
	_, err = ioutil.ReadAll(&LimitedReader{R: r, N: 2})
	assert.Error(t, err, "read exceeds the defined limit")
}
