package formatter

import "strings"

// Common header constants
const (
	CreatedSinceHeader = "CREATED"
	CreatedAtHeader    = "CREATED AT"
	SizeHeader         = "SIZE"
	LabelsHeader       = "LABELS"
	NameHeader         = "NAME"
	DescriptionHeader  = "DESCRIPTION"
	DriverHeader       = "DRIVER"
	ScopeHeader        = "SCOPE"
	StateHeader        = "STATE"
	StatusHeader       = "STATUS"
	PortsHeader        = "PORTS"
	ImageHeader        = "IMAGE"
	ContainerIDHeader  = "CONTAINER ID"
)

// SubContext defines what Context implementation should provide
type SubContext interface {
	FullHeader() interface{}
}

// SubHeaderContext is a map destined to formatter header (table format)
type SubHeaderContext map[string]string

// Label returns the header label for the specified string
func (c SubHeaderContext) Label(name string) string {
	n := strings.Split(name, ".")
	r := strings.NewReplacer("-", " ", "_", " ")
	h := r.Replace(n[len(n)-1])

	return h
}

// HeaderContext provides the subContext interface for managing headers
type HeaderContext struct {
	Header interface{}
}

// FullHeader returns the header as an interface
func (c *HeaderContext) FullHeader() interface{} {
	return c.Header
}

func stripNamePrefix(ss []string) []string {
	sss := make([]string, len(ss))
	for i, s := range ss {
		sss[i] = s[1:]
	}

	return sss
}
