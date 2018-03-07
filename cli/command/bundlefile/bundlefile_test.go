package bundlefile

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	is "github.com/gotestyourself/gotestyourself/assert/cmp"
)

func TestLoadFileV01Success(t *testing.T) {
	reader := strings.NewReader(`{
		"Version": "0.1",
		"Services": {
			"redis": {
				"Image": "redis@sha256:4b24131101fa0117bcaa18ac37055fffd9176aa1a240392bb8ea85e0be50f2ce",
				"Networks": ["default"]
			},
			"web": {
				"Image": "dockercloud/hello-world@sha256:fe79a2cfbd17eefc344fb8419420808df95a1e22d93b7f621a7399fd1e9dca1d",
				"Networks": ["default"],
				"User": "web"
			}
		}
	}`)

	bundle, err := LoadFile(reader)
	assert.NilError(t, err)
	assert.Check(t, is.Equal("0.1", bundle.Version))
	assert.Check(t, is.Len(bundle.Services, 2))
}

func TestLoadFileSyntaxError(t *testing.T) {
	reader := strings.NewReader(`{
		"Version": "0.1",
		"Services": unquoted string
	}`)

	_, err := LoadFile(reader)
	assert.Error(t, err, "JSON syntax error at byte 37: invalid character 'u' looking for beginning of value")
}

func TestLoadFileTypeError(t *testing.T) {
	reader := strings.NewReader(`{
		"Version": "0.1",
		"Services": {
			"web": {
				"Image": "redis",
				"Networks": "none"
			}
		}
	}`)

	_, err := LoadFile(reader)
	assert.Error(t, err, "Unexpected type at byte 94. Expected []string but received string.")
}

func TestPrint(t *testing.T) {
	var buffer bytes.Buffer
	bundle := &Bundlefile{
		Version: "0.1",
		Services: map[string]Service{
			"web": {
				Image:   "image",
				Command: []string{"echo", "something"},
			},
		},
	}
	assert.Check(t, Print(&buffer, bundle))
	output := buffer.String()
	assert.Check(t, is.Contains(output, "\"Image\": \"image\""))
	assert.Check(t, is.Contains(output,
		`"Command": [
                "echo",
                "something"
            ]`))
}
