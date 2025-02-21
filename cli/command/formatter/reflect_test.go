// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.22

package formatter // import "docker.com/cli/v28/cli/command/formatter"

import (
	"reflect"
	"testing"
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
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dummyExpected, m) {
		t.Fatalf("expected %+v, got %+v",
			dummyExpected, m)
	}
}

func TestMarshalMapBad(t *testing.T) {
	if _, err := marshalMap(nil); err == nil {
		t.Fatal("expected an error (argument is nil)")
	}
	if _, err := marshalMap(dummy{}); err == nil {
		t.Fatal("expected an error (argument is non-pointer)")
	}
	x := 42
	if _, err := marshalMap(&x); err == nil {
		t.Fatal("expected an error (argument is a pointer to non-struct)")
	}
}
