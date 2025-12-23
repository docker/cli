// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package formatter

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

type dummy struct{}

func (*dummy) Func1() string {
	return "Func1"
}

func (*dummy) func2() string { //nolint:unused
	return "func2(should not be marshalled)"
}

func (*dummy) Func3() (string, int) {
	return "Func3(should not be marshalled)", -42
}

func (*dummy) Func4() int {
	return 4
}

type dummyType string

func (*dummy) Func5() dummyType {
	return "Func5"
}

func (*dummy) FullHeader() string {
	return "FullHeader(should not be marshalled)"
}

var dummyExpected = map[string]any{
	"Func1": "Func1",
	"Func4": 4,
	"Func5": dummyType("Func5"),
}

func TestMarshalMap(t *testing.T) {
	d := dummy{}
	m, err := marshalMap(&d)
	assert.NilError(t, err)
	assert.Check(t, is.DeepEqual(m, dummyExpected))
}

func TestMarshalMapBad(t *testing.T) {
	_, err := marshalMap(nil)
	assert.Check(t, is.Error(err, "expected a pointer to a struct, got invalid"), "expected an error (argument is nil)")

	_, err = marshalMap(dummy{})
	assert.Check(t, is.Error(err, "expected a pointer to a struct, got struct"), "expected an error (argument is non-pointer)")

	x := 42
	_, err = marshalMap(&x)
	assert.Check(t, is.Error(err, "expected a pointer to a struct, got a pointer to int"), "expected an error (argument is a pointer to non-struct)")
}
