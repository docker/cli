package json

import (
	"io"

	"encoding/json"
	"github.com/pkg/errors"
)

type specedFields struct {
	Spec namedFields
}

// Speced objects are used to partially decode a json object that has a Spec with
// a name and labels. The rest of the structure is untouched.
type Speced struct {
	specedFields
	raw map[string]interface{}
}

func (n *Speced) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &n.specedFields); err != nil {
		return errors.Wrapf(err, "failed to unmarshal Speced")
	}
	return json.Unmarshal(data, &n.raw)
}

func (n *Speced) MarshalJSON() ([]byte, error) {
	asMap := map[string]interface{}(n.raw)
	if asMap["Spec"] == nil {
		asMap["Spec"] = make(map[string]interface{})
	}
	innerMap, ok := asMap["Spec"].(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("unexpected Spec format")
	}
	innerMap["Name"] = n.Spec.Name
	innerMap["Labels"] = n.Spec.Labels
	return json.Marshal(&asMap)
}

// DecodeSpecedObject decodes a json request body and returns a list of Speced
// objects. Both Volumes and Networks can be decoded this way.
func DecodeSpeceds(body io.Reader) ([]Speced, error) {
	var speced []Speced
	return speced, decode(body, &speced)
}

// DecodeSpecedObject decodes a json request body and returns a Speced object.
// Both Volumes and Networks can be decoded this way.
func DecodeSpeced(body io.Reader) (Speced, error) {
	var speced Speced
	return speced, decode(body, &speced)
}
