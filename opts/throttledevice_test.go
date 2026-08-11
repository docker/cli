package opts

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestValidateThrottleBpsDevice(t *testing.T) {
	tests := []struct {
		doc          string
		input        string
		expectedErr  string
		expectedPath string
		expectedRate uint64
	}{
		{doc: "plain integer", input: "/dev/sda:1000", expectedPath: "/dev/sda", expectedRate: 1000},
		{doc: "with unit", input: "/dev/sda:1mb", expectedPath: "/dev/sda", expectedRate: 1048576},
		{doc: "zero", input: "/dev/sda:0", expectedPath: "/dev/sda", expectedRate: 0},
		{doc: "missing colon", input: "/dev/sda", expectedErr: "bad format: /dev/sda"},
		{doc: "empty device", input: ":1mb", expectedErr: "bad format: :1mb"},
		{doc: "missing /dev/ prefix", input: "sda:1mb", expectedErr: "bad format for device path: sda:1mb"},
		{doc: "non-numeric rate", input: "/dev/sda:foo", expectedErr: "invalid rate for device"},
	}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			v, err := ValidateThrottleBpsDevice(tc.input)
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
				assert.Check(t, is.Nil(v))
				return
			}
			assert.NilError(t, err)
			assert.Check(t, is.Equal(v.Path, tc.expectedPath))
			assert.Check(t, is.Equal(v.Rate, tc.expectedRate))
		})
	}
}

func TestValidateThrottleIOpsDevice(t *testing.T) {
	tests := []struct {
		doc          string
		input        string
		expectedErr  string
		expectedPath string
		expectedRate uint64
	}{
		{doc: "valid integer", input: "/dev/sda:100", expectedPath: "/dev/sda", expectedRate: 100},
		{doc: "fractional rejected", input: "/dev/sda:1.5", expectedErr: "invalid rate for device"},
		{doc: "negative rejected", input: "/dev/sda:-5", expectedErr: "invalid rate for device"},
		{doc: "unit suffix rejected (iops are integers)", input: "/dev/sda:1mb", expectedErr: "invalid rate for device"},
		{doc: "missing /dev/ prefix", input: "sda:100", expectedErr: "bad format for device path: sda:100"},
	}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			v, err := ValidateThrottleIOpsDevice(tc.input)
			if tc.expectedErr != "" {
				assert.Check(t, is.ErrorContains(err, tc.expectedErr))
				assert.Check(t, is.Nil(v))
				return
			}
			assert.NilError(t, err)
			assert.Check(t, is.Equal(v.Path, tc.expectedPath))
			assert.Check(t, is.Equal(v.Rate, tc.expectedRate))
		})
	}
}

func TestThrottledeviceOptSetGetList(t *testing.T) {
	opt := NewThrottledeviceOpt(ValidateThrottleBpsDevice)
	assert.NilError(t, opt.Set("/dev/sda:1mb"))
	assert.NilError(t, opt.Set("/dev/sdb:2mb"))

	list := opt.GetList()
	assert.Assert(t, is.Len(list, 2))
	assert.Check(t, is.Equal(list[0].Path, "/dev/sda"))
	assert.Check(t, is.Equal(list[0].Rate, uint64(1048576)))
	assert.Check(t, is.Equal(list[1].Path, "/dev/sdb"))
	assert.Check(t, is.Equal(list[1].Rate, uint64(2097152)))

	assert.Check(t, is.ErrorContains(opt.Set("/dev/sdc:bad"), "invalid rate for device"))
	assert.Check(t, is.Equal(opt.Type(), "list"))
}
