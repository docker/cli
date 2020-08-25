package opts

import (
	"strconv"
	"testing"

	"gotest.tools/v3/assert"
)

func TestNormalizeCapability(t *testing.T) {
	tests := []struct{ in, out string }{
		{in: "ALL", out: "ALL"},
		{in: "FOO", out: "CAP_FOO"},
		{in: "CAP_FOO", out: "CAP_FOO"},
		{in: "CAPFOO", out: "CAP_CAPFOO"},

		// case-insensitive handling
		{in: "aLl", out: "ALL"},
		{in: "foO", out: "CAP_FOO"},
		{in: "cAp_foO", out: "CAP_FOO"},

		// white space handling. strictly, these could be considered "invalid",
		// but are a likely situation, so handling these for now.
		{in: "  ALL  ", out: "ALL"},
		{in: "  FOO  ", out: "CAP_FOO"},
		{in: "  CAP_FOO  ", out: "CAP_FOO"},
		{in: " 	 ALL 	 ", out: "ALL"},
		{in: " 	 FOO 	 ", out: "CAP_FOO"},
		{in: " 	 CAP_FOO 	 ", out: "CAP_FOO"},

		// weird values: no validation takes place currently, so these
		// are handled same as values above; we could consider not accepting
		// these in future
		{in: "SOME CAP", out: "CAP_SOME CAP"},
		{in: "_FOO", out: "CAP__FOO"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, NormalizeCapability(tc.in), tc.out)
		})
	}
}

func TestEffectiveCapAddCapDrop(t *testing.T) {
	type caps struct {
		add, drop []string
	}

	tests := []struct {
		in, out caps
	}{
		{
			in: caps{
				add:  []string{"one", "two"},
				drop: []string{"one", "two"},
			},
			out: caps{
				add: []string{"CAP_ONE", "CAP_TWO"},
			},
		},
		{
			in: caps{
				add:  []string{"CAP_ONE", "cap_one", "CAP_TWO"},
				drop: []string{"one", "cap_two"},
			},
			out: caps{
				add: []string{"CAP_ONE", "CAP_TWO"},
			},
		},
		{
			in: caps{
				add:  []string{"CAP_ONE", "CAP_TWO"},
				drop: []string{"CAP_ONE", "CAP_THREE"},
			},
			out: caps{
				add:  []string{"CAP_ONE", "CAP_TWO"},
				drop: []string{"CAP_THREE"},
			},
		},
		{
			in: caps{
				add:  []string{"ALL"},
				drop: []string{"CAP_ONE", "CAP_TWO"},
			},
			out: caps{
				add:  []string{"ALL"},
				drop: []string{"CAP_ONE", "CAP_TWO"},
			},
		},
		{
			in: caps{
				add: []string{"ALL", "CAP_ONE"},
			},
			out: caps{
				add: []string{"ALL"},
			},
		},
		{
			in: caps{
				drop: []string{"ALL", "CAP_ONE"},
			},
			out: caps{
				drop: []string{"ALL"},
			},
		},
	}

	for i, tc := range tests {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			add, drop := EffectiveCapAddCapDrop(tc.in.add, tc.in.drop)
			assert.DeepEqual(t, add, tc.out.add)
			assert.DeepEqual(t, drop, tc.out.drop)

		})
	}
}
