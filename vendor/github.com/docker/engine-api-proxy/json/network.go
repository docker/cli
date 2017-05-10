package json

import (
	"io"

	"encoding/json"
	"github.com/pkg/errors"
)

type connectedFields struct {
	Container string
}

// Connected objects are used to partially decode a json object that has a
// container name and other fields
type Connected struct {
	connectedFields
	raw map[string]interface{}
}

func (n *Connected) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &n.connectedFields); err != nil {
		return errors.Wrapf(err, "failed to unmarshal Connected")
	}
	return json.Unmarshal(data, &n.raw)
}

func (n *Connected) MarshalJSON() ([]byte, error) {
	asMap := map[string]interface{}(n.raw)
	asMap["Container"] = n.Container
	return json.Marshal(&asMap)
}

// DecodeNetworkConnect decodes a json request body and returns a network
// connect body
func DecodeNetworkConnect(body io.Reader) (Connected, error) {
	var net Connected
	return net, decode(body, &net)
}
