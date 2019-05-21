package store

import (
	"io"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestLimitReaderReadAll(t *testing.T) {
	var r io.Reader
	r = strings.NewReader("Reader")

	_, err := ioutil.ReadAll(r)
	assert.NilError(t, err)

	r = strings.NewReader("Reader")
	_, err = LimitedReadAll(r, 4)
	assert.ErrorType(t, err, reflect.TypeOf(&ReadExceedsLimitError{limit: 4}))
}
