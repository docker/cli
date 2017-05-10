package json

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

// Encode json encodes obj and returns it as a ReadCloser
func Encode(obj interface{}) (int, io.ReadCloser, error) {
	var b bytes.Buffer
	if obj == nil {
		return 0, ioutil.NopCloser(&b), nil
	}

	encoder := json.NewEncoder(&b)
	if err := encoder.Encode(&obj); err != nil {
		log.Warnf("Failed to re-encode %T: %s\n", obj, err)
		return -1, nil, err
	}
	return b.Len(), ioutil.NopCloser(&b), nil
}

// EncodeBody json encodes obj and returns a request with the object in the body
func EncodeBody(obj interface{}, req *http.Request) (*http.Request, error) {
	var size int
	var err error
	size, req.Body, err = Encode(obj)
	req.ContentLength = int64(size)
	return req, err
}

func decode(body io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(obj); err != nil {
		log.Warnf("Failed to decode %T: %s\n", obj, err)
		return err
	}
	return nil
}

type namedFields struct {
	Name   string
	Labels map[string]string
}

// Named objects are used to partially decode a json object that has a name and
// labels. The rest of the structure is untouched.
type Named struct {
	namedFields
	raw map[string]interface{}
}

func (n *Named) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &n.namedFields); err != nil {
		return errors.Wrapf(err, "failed to unmarshal Named")
	}
	return json.Unmarshal(data, &n.raw)
}

func (n *Named) MarshalJSON() ([]byte, error) {
	asMap := map[string]interface{}(n.raw)
	asMap["Name"] = n.Name
	asMap["Labels"] = n.Labels
	return json.Marshal(&asMap)
}

// DecodeNamedObject decodes a json request body and returns a list of Named
// objects. Both Volumes and Networks can be decoded this way.
func DecodeNameds(body io.Reader) ([]Named, error) {
	var named []Named
	return named, decode(body, &named)
}

// DecodeNamedObject decodes a json request body and returns a Named object.
// Both Volumes and Networks can be decoded this way.
func DecodeNamed(body io.Reader) (Named, error) {
	var named Named
	return named, decode(body, &named)
}
