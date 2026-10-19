// Package prompt provides utilities to prompt the user for input.
//
// It is a trimmed-down copy of the CLI's "internal/prompt" package, which
// this module cannot import: the root module carries a major-version suffix
// in its path, which puts its "internal/" packages out of reach here.
package prompt

import (
	"bufio"
	"context"
	"io"
	"os"
	"runtime"
	"strings"
)

const ErrTerminated cancelledErr = "prompt terminated"

type cancelledErr string

func (e cancelledErr) Error() string {
	return string(e)
}

func (cancelledErr) Cancelled() {}

// Confirm requests and checks confirmation from the user.
//
// It displays the provided message followed by "[y/N]". If the user
// input 'y' or 'Y' it returns true otherwise false. If no message is provided,
// "Are you sure you want to proceed? [y/N] " will be used instead.
//
// It returns false with an [ErrTerminated] if the user terminates
// the CLI with SIGINT or SIGTERM while the prompt is active. If the prompt
// returns an error, the caller should close the [io.Reader] used for the prompt
// and propagate the error up the stack to prevent the background goroutine
// from blocking indefinitely.
func Confirm(ctx context.Context, in io.Reader, out io.Writer, message string) (bool, error) {
	if message == "" {
		message = "Are you sure you want to proceed?"
	}
	message += " [y/N] "

	_, _ = out.Write([]byte(message))

	// On Windows, force the use of the regular OS stdin stream.
	//
	// StdStreams() may wrap stdin with windowsconsole.NewAnsiReader to
	// emulate VT input on consoles that do not support it natively, but
	// that wrapper has historically caused interactive prompts to hang
	// or behave incorrectly.
	//
	// See:
	// - https://github.com/moby/moby/issues/14336
	// - https://github.com/moby/moby/issues/14210
	// - https://github.com/moby/moby/pull/17738
	if runtime.GOOS == "windows" {
		in = os.Stdin
	}

	result := make(chan bool)

	go func() {
		var res bool
		scanner := bufio.NewScanner(in)
		if scanner.Scan() {
			answer := strings.TrimSpace(scanner.Text())
			if strings.EqualFold(answer, "y") {
				res = true
			}
		}
		result <- res
	}()

	select {
	case <-ctx.Done():
		_, _ = out.Write([]byte("\n"))
		return false, ErrTerminated
	case r := <-result:
		return r, nil
	}
}
