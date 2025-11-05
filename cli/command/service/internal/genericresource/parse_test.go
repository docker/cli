package genericresource

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseDiscrete(t *testing.T) {
	res, err := ParseCmd("apple=3")
	assert.NilError(t, err)
	assert.Equal(t, len(res), 1)

	apples := GetResource("apple", res)
	assert.Equal(t, len(apples), 1)
	if apples[0].DiscreteResourceSpec == nil {
		t.Fatalf("expected discrete resource spec, got nil")
	}
	assert.Equal(t, apples[0].DiscreteResourceSpec.Value, int64(3))

	_, err = ParseCmd("apple=3\napple=4")
	assert.Assert(t, err != nil)

	_, err = ParseCmd("apple=3,apple=4")
	assert.Assert(t, err != nil)

	_, err = ParseCmd("apple=-3")
	assert.Assert(t, err != nil)
}

func TestParseStr(t *testing.T) {
	res, err := ParseCmd("orange=red,orange=green,orange=blue")
	assert.NilError(t, err)
	assert.Equal(t, len(res), 3)

	oranges := GetResource("orange", res)
	assert.Equal(t, len(oranges), 3)
	for _, k := range []string{"red", "green", "blue"} {
		assert.Assert(t, HasResource(NewString("orange", k), oranges))
	}
}

func TestParseDiscreteAndStr(t *testing.T) {
	res, err := ParseCmd("orange=red,orange=green,orange=blue,apple=3")
	assert.NilError(t, err)
	assert.Equal(t, len(res), 4)

	oranges := GetResource("orange", res)
	assert.Equal(t, len(oranges), 3)
	for _, k := range []string{"red", "green", "blue"} {
		assert.Assert(t, HasResource(NewString("orange", k), oranges))
	}

	apples := GetResource("apple", res)
	assert.Equal(t, len(apples), 1)
	if apples[0].DiscreteResourceSpec == nil {
		t.Fatalf("expected discrete resource spec, got nil")
	}
	assert.Equal(t, apples[0].DiscreteResourceSpec.Value, int64(3))
}

func TestParseMixedForSameKindFails(t *testing.T) {
	_, err := ParseCmd("gpu=fast,gpu=slow,gpu=2")
	assert.Assert(t, err != nil)
}
