package hint

import (
	"errors"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

func TestWrap_NilError(t *testing.T) {
	assert.Assert(t, Wrap(nil, "irrelevant") == nil)
}

func TestWrap_PreservesMessage(t *testing.T) {
	err := Wrap(errors.New("bad input"), "Try --help.")
	assert.Equal(t, err.Error(), "bad input")
}

func TestWrap_HintReadable(t *testing.T) {
	err := Wrap(errors.New("bad input"), "Try --help.")

	var h Hinter
	assert.Assert(t, errors.As(err, &h))
	assert.Equal(t, h.Hint(), "Try --help.")
}

func TestOf_FindsHint(t *testing.T) {
	base := Wrap(errors.New("bad input"), "Try --help.")
	wrapped := fmt.Errorf("context: %w", base)

	assert.Equal(t, Of(wrapped), "Try --help.")
}

func TestOf_NoHint(t *testing.T) {
	assert.Equal(t, Of(errors.New("plain")), "")
	assert.Equal(t, Of(nil), "")
}

func TestOf_FindsHintInJoinedError(t *testing.T) {
	err := errors.Join(errors.New("plain"), Wrap(errors.New("bad input"), "Try --help."))

	assert.Equal(t, Of(err), "Try --help.")
}

func TestUnwrap(t *testing.T) {
	base := errors.New("bad input")
	err := Wrap(base, "Try --help.")
	assert.Assert(t, errors.Is(err, base))
}
