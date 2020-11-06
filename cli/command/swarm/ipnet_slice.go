package swarm

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"strings"
)

// -- ipNetSlice Value
type ipNetSliceValue struct {
	value   *[]net.IPNet
	changed bool
}

func newIPNetSliceValue(val []net.IPNet, p *[]net.IPNet) *ipNetSliceValue {
	ipnsv := new(ipNetSliceValue)
	ipnsv.value = p
	*ipnsv.value = val
	return ipnsv
}

// Set converts, and assigns, the comma-separated IPNet argument string representation as the []net.IPNet value of this flag.
// If Set is called on a flag that already has a []net.IPNet assigned, the newly converted values will be appended.
func (s *ipNetSliceValue) Set(val string) error {

	// remove all quote characters
	rmQuote := strings.NewReplacer(`"`, "", `'`, "", "`", "")

	// read flag arguments with CSV parser
	ipNetStrSlice, err := readAsCSV(rmQuote.Replace(val))
	if err != nil && err != io.EOF {
		return err
	}

	// parse ip values into slice
	out := make([]net.IPNet, 0, len(ipNetStrSlice))
	for _, ipNetStr := range ipNetStrSlice {
		_, n, err := net.ParseCIDR(strings.TrimSpace(ipNetStr))
		if err != nil {
			return fmt.Errorf("invalid string being converted to CIDR: %s", ipNetStr)
		}
		out = append(out, *n)
	}

	if !s.changed {
		*s.value = out
	} else {
		*s.value = append(*s.value, out...)
	}

	s.changed = true

	return nil
}

// Type returns a string that uniquely represents this flag's type.
func (s *ipNetSliceValue) Type() string {
	return "ipNetSlice"
}

// String defines a "native" format for this net.IPNet slice flag value.
func (s *ipNetSliceValue) String() string {

	ipNetStrSlice := make([]string, len(*s.value))
	for i, n := range *s.value {
		ipNetStrSlice[i] = n.String()
	}

	out, _ := writeAsCSV(ipNetStrSlice)
	return "[" + out + "]"
}

func readAsCSV(val string) ([]string, error) {
	if val == "" {
		return []string{}, nil
	}
	stringReader := strings.NewReader(val)
	csvReader := csv.NewReader(stringReader)
	return csvReader.Read()
}

func writeAsCSV(vals []string) (string, error) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	err := w.Write(vals)
	if err != nil {
		return "", err
	}
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n"), nil
}
