package streams

import (
	"errors"
	"io"
	"runtime"

	"github.com/moby/term"
)

// In is an input stream to read user input. It implements [io.ReadCloser]
// with additional utilities, such as putting the terminal in raw mode.
type In struct {
	in io.ReadCloser
	cs commonStream
}

// NewIn returns a new [In] from an [io.ReadCloser].
func NewIn(in io.ReadCloser) *In {
	return &In{
		in: in,
		cs: newCommonStream(in),
	}
}

// FD returns the file descriptor number for this stream.
func (i *In) FD() uintptr {
	return i.cs.fd
}

// Read implements the [io.Reader] interface.
func (i *In) Read(p []byte) (int, error) {
	return i.in.Read(p)
}

// Close implements the [io.Closer] interface.
func (i *In) Close() error {
	return i.in.Close()
}

// IsTerminal returns whether this stream is connected to a terminal.
func (i *In) IsTerminal() bool {
	return i.cs.isTerminal()
}

// SetRawTerminal sets raw mode on the input terminal. It is a no-op if In
// is not a TTY, or if the "NORAW" environment variable is set to a non-empty
// value.
func (i *In) SetRawTerminal() error {
	return i.cs.setRawTerminal(term.SetRawTerminal)
}

// RestoreTerminal restores the terminal state if SetRawTerminal succeeded earlier.
func (i *In) RestoreTerminal() {
	i.cs.restoreTerminal()
}

// CheckTty checks if we are trying to attach to a container TTY
// from a non-TTY client input stream, and if so, returns an error.
func (i *In) CheckTty(attachStdin, ttyMode bool) error {
	// In order to attach to a container tty, input stream for the client must
	// be a tty itself: redirecting or piping the client standard input is
	// incompatible with `docker run -t`, `docker exec -t` or `docker attach`.
	if ttyMode && attachStdin && !i.cs.isTerminal() {
		const eText = "the input device is not a TTY"
		if runtime.GOOS == "windows" {
			return errors.New(eText + ".  If you are using mintty, try prefixing the command with 'winpty'")
		}
		return errors.New(eText)
	}
	return nil
}

// SetIsTerminal overrides whether a terminal is connected. It is used to
// override this property in unit-tests, and should not be depended on for
// other purposes.
func (i *In) SetIsTerminal(isTerminal bool) {
	i.cs.setIsTerminal(isTerminal)
}
