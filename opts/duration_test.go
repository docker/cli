package opts

import (
	"testing"
	"time"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestDurationOptString(t *testing.T) {
	dur := time.Duration(300 * 10e8)
	duration := DurationOpt{value: &dur}
	assert.Check(t, is.Equal("5m0s", duration.String()))
}

func TestDurationOptSetAndValue(t *testing.T) {
	var duration DurationOpt
	assert.Check(t, duration.Set("300s"))
	assert.Check(t, is.Equal(time.Duration(300*10e8), *duration.Value()))
	assert.Check(t, duration.Set("-300s"))
	assert.Check(t, is.Equal(time.Duration(-300*10e8), *duration.Value()))
}

func TestPositiveDurationOptSetAndValue(t *testing.T) {
	var duration PositiveDurationOpt
	assert.Check(t, duration.Set("300s"))
	assert.Check(t, is.Equal(time.Duration(300*10e8), *duration.Value()))
	assert.Check(t, is.Error(duration.Set("-300s"), "duration cannot be negative"))
}
