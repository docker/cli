package opts

import (
	"bytes"
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseKeyValueFile(t *testing.T) {
	b := []byte(`
FOO=BAR
ZOT`)

	vars := map[string]string{
		"ZOT": "QIX",
	}
	lookupFn := func(s string) (string, bool) {
		v, ok := vars[s]
		return v, ok
	}

	got, err := ParseKeyValueFile(bytes.NewReader(b), "(inlined)", lookupFn)
	assert.NilError(t, err)
	assert.DeepEqual(t, got, []string{"FOO=BAR", "ZOT=QIX"})
}
