package json

import (
	"io"

	"github.com/docker/docker/api/types/volume"
)

// DecodeVolumeList decodes a json request body and returns a list of volumes
// summaries
func DecodeVolumeList(body io.Reader) (volume.VolumesListOKBody, error) {
	var parsedBody volume.VolumesListOKBody
	return parsedBody, decode(body, &parsedBody)
}
