package streams

import (
	"os"

	"github.com/moby/term"
	"github.com/sirupsen/logrus"
)

func newCommonStream(stream any) commonStream {
	fd, tty := term.GetFdInfo(stream)
	return commonStream{
		fd:  fd,
		tty: tty,
	}
}

type commonStream struct {
	fd    uintptr
	tty   bool
	state *term.State
}

// FD returns the file descriptor number for this stream.
func (s *commonStream) FD() uintptr { return s.fd }

// isTerminal returns whether this stream is connected to a terminal.
func (s *commonStream) isTerminal() bool { return s.tty }

// setIsTerminal overrides whether a terminal is connected for testing.
func (s *commonStream) setIsTerminal(isTerminal bool) { s.tty = isTerminal }

// restoreTerminal restores the terminal state if SetRawTerminal succeeded earlier.
func (s *commonStream) restoreTerminal() {
	if s.state != nil {
		_ = term.RestoreTerminal(s.fd, s.state)
	}
}

func (s *commonStream) setRawTerminal(setter func(uintptr) (*term.State, error)) error {
	if !s.tty || os.Getenv("NORAW") != "" {
		return nil
	}
	state, err := setter(s.fd)
	if err != nil {
		return err
	}
	s.state = state
	return nil
}

func (s *commonStream) terminalSize() (height uint, width uint) {
	if !s.tty {
		return 0, 0
	}
	ws, err := term.GetWinsize(s.fd)
	if err != nil {
		logrus.WithError(err).Debug("Error getting TTY size")
		if ws == nil {
			return 0, 0
		}
	}
	return uint(ws.Height), uint(ws.Width)
}
