package json

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Stub struct {
	Foo string
}

func TestEncodeObject(t *testing.T) {
	stub := Stub{Foo: "ok"}
	expectedSize := 13

	size, reader, err := Encode(&stub)
	assert.NoError(t, err)
	assert.Equal(t, expectedSize, size)

	buf, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, expectedSize, len(buf))
	assert.Equal(t, `{"Foo":"ok"}`+"\n", string(buf))
}

func TestEncodeWithNil(t *testing.T) {
	size, reader, err := Encode(nil)
	assert.NoError(t, err)
	assert.Equal(t, size, 0)

	buf, err := ioutil.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(buf))
}
