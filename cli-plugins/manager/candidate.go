package manager

import (
	"os/exec"

	"github.com/docker/cli/cli-plugins/metadata"
)

// Candidate represents a possible plugin candidate, for mocking purposes
type Candidate interface {
	Path() string
	Metadata() ([]byte, error)
}

type candidate struct {
	path string
}

func (c *candidate) Path() string {
	return c.path
}

func (c *candidate) Metadata() ([]byte, error) {
	return exec.Command(c.path, metadata.MetadataSubcommandName).Output() // #nosec G204 -- ignore "Subprocess launched with a potential tainted input or cmd arguments"
}
