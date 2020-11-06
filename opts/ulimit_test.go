package opts

import (
	"testing"

	"github.com/docker/go-units"
	"gotest.tools/v3/assert"
)

func TestUlimitOpt(t *testing.T) {
	ulimitMap := map[string]*units.Ulimit{
		"nofile": {Name: "nofile", Hard: 1024, Soft: 512},
	}

	ulimitOpt := NewUlimitOpt(&ulimitMap)

	expected := "[nofile=512:1024]"
	assert.Equal(t, ulimitOpt.String(), expected)

	// Valid ulimit append to opts
	err := ulimitOpt.Set("core=1024:1024")
	assert.NilError(t, err)

	err = ulimitOpt.Set("nofile")
	assert.ErrorContains(t, err, "invalid ulimit argument")

	// Invalid ulimit type returns an error and do not append to opts
	err = ulimitOpt.Set("notavalidtype=1024:1024")
	assert.ErrorContains(t, err, "invalid ulimit type")

	expected = "[core=1024:1024 nofile=512:1024]"
	assert.Equal(t, ulimitOpt.String(), expected)

	// And test GetList
	ulimits := ulimitOpt.GetList()
	assert.Equal(t, len(ulimits), 2)
}

func TestUlimitOptSorting(t *testing.T) {
	ulimitOpt := NewUlimitOpt(&map[string]*units.Ulimit{
		"nofile": {Name: "nofile", Hard: 1024, Soft: 512},
		"core":   {Name: "core", Hard: 1024, Soft: 1024},
	})

	expected := []*units.Ulimit{
		{Name: "core", Hard: 1024, Soft: 1024},
		{Name: "nofile", Hard: 1024, Soft: 512},
	}

	ulimits := ulimitOpt.GetList()
	assert.DeepEqual(t, ulimits, expected)

	assert.Equal(t, ulimitOpt.String(), "[core=1024:1024 nofile=512:1024]")
}
