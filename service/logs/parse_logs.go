/*Package logs contains tools for parsing docker log lines.
 */
package logs

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// ParseLogDetails parses a string of key value pairs in the form
// "k=v,l=w", where the keys and values are url query escaped, and each pair
// is separated by a comma. Returns a map of the key value pairs on success,
// and an error if the details string is not in a valid format.
//
// The details string encoding is implemented in
// github.com/moby/moby/api/server/httputils/write_log_stream.go
func ParseLogDetails(details string) (map[string]string, error) {
	pairs := strings.Split(details, ",")
	detailsMap := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		k, v, ok := strings.Cut(pair, "=")
		if !ok || k == "" {
			// missing equal sign, or no key.
			return nil, errors.New("invalid details format")
		}
		var err error
		k, err = url.QueryUnescape(k)
		if err != nil {
			return nil, err
		}
		v, err = url.QueryUnescape(v)
		if err != nil {
			return nil, err
		}
		detailsMap[k] = v
	}
	return detailsMap, nil
}
