package store

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func TestLimitReaderReadAll(t *testing.T) {
	r := strings.NewReader("Reader")

	_, err := ioutil.ReadAll(r)
	assert.NilError(t, err)

	r = strings.NewReader("Reader")
	_, err = LimitedReadAll(r, 4)
	assert.Error(t, err, fmt.Sprintf("read exceeds the defined limit of %d on the reader", 4))
}
