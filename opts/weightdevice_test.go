package opts

import (
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestValidateWeightDevice(t *testing.T) {
	tests := []struct {
		doc            string
		input          string
		expectedErr    string
		expectedPath   string
		expectedWeight uint16
	}{
		{doc: "valid minimum", input: "/dev/sda:10", expectedPath: "/dev/sda", expectedWeight: 10},
		{doc: "valid maximum", input: "/dev/sda:1000", expectedPath: "/dev/sda", expectedWeight: 1000},
		{doc: "zero is accepted (unset)", input: "/dev/sda:0", expectedPath: "/dev/sda", expectedWeight: 0},
		{doc: "below minimum", input: "/dev/sda:9", expectedErr: "invalid weight for device: /dev/sda:9"},
		{doc: "above maximum", input: "/dev/sda:1001", expectedErr: "invalid weight for device: /dev/sda:1001"},
		{doc: "overflows uint16", input: "/dev/sda:70000", expectedErr: "invalid weight for device: /dev/sda:70000"},
		{doc: "missing colon", input: "/dev/sda", expectedErr: "bad format: /dev/sda"},
		{doc: "empty device", input: ":100", expectedErr: "bad format: :100"},
		{doc: "missing /dev/ prefix", input: "sda:100", expectedErr: "bad format for device path: sda:100"},
	}
	for _, tc := range tests {
		t.Run(tc.doc, func(t *testing.T) {
			v, err := ValidateWeightDevice(tc.input)
			if tc.expectedErr != "" {
				assert.Check(t, is.Error(err, tc.expectedErr))
				assert.Check(t, is.Nil(v))
				return
			}
			assert.NilError(t, err)
			assert.Check(t, is.Equal(v.Path, tc.expectedPath))
			assert.Check(t, is.Equal(v.Weight, tc.expectedWeight))
		})
	}
}

func TestWeightdeviceOptSetGetList(t *testing.T) {
	opt := NewWeightdeviceOpt(ValidateWeightDevice)
	assert.NilError(t, opt.Set("/dev/sda:100"))
	assert.NilError(t, opt.Set("/dev/sdb:200"))

	list := opt.GetList()
	assert.Assert(t, is.Len(list, 2))
	assert.Check(t, is.Equal(list[0].Path, "/dev/sda"))
	assert.Check(t, is.Equal(list[0].Weight, uint16(100)))
	assert.Check(t, is.Equal(list[1].Path, "/dev/sdb"))
	assert.Check(t, is.Equal(list[1].Weight, uint16(200)))

	assert.Check(t, is.Error(opt.Set("/dev/sdc:1"), "invalid weight for device: /dev/sdc:1"))
	assert.Check(t, is.Equal(opt.Type(), "list"))
}
