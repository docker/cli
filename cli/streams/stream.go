package streams

import (
	"os"

	"github.com/moby/term"
)

type commonStream struct {
	fd         uintptr
	isTerminal bool
	state      *term.State
}

// FD returns the file descriptor number for this stream.
func (s *commonStream) FD() uintptr {
	return s.fd
}

// IsTerminal returns true if this stream is connected to a terminal.
func (s *commonStream) IsTerminal() bool {
	return s.isTerminal
}

// RestoreTerminal restores the terminal state if SetRawTerminal succeeded earlier.
func (s *commonStream) RestoreTerminal() {
	if s.state != nil {
		_ = term.RestoreTerminal(s.fd, s.state)
	}
}

func (s *commonStream) setRawTerminal(setter func(uintptr) (*term.State, error)) error {
	if !s.isTerminal || os.Getenv("NORAW") != "" {
		return nil
	}
	state, err := setter(s.fd)
	if err != nil {
		return err
	}
	s.state = state
	return nil
}

// SetIsTerminal overrides whether a terminal is connected. It is used to
// override this property in unit-tests, and should not be depended on for
// other purposes.
func (s *commonStream) SetIsTerminal(isTerminal bool) {
	s.isTerminal = isTerminal
}
