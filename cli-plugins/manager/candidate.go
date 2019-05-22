package manager

import (
	"os/exec"
)

// Candidate represents a possible plugin candidate, for mocking purposes
type Candidate interface {
	Path() string
	Metadata() ([]byte, error)
	Experimental() bool
}

type candidate struct {
	path         string
	experimental bool
}

func (c *candidate) Path() string {
	return c.path
}

func (c *candidate) Experimental() bool {
	return c.experimental
}

func (c *candidate) Metadata() ([]byte, error) {
	return exec.Command(c.path, MetadataSubcommandName).Output()
}
