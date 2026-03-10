package hooks

import (
	"fmt"
	"io"

	"github.com/morikuni/aec"
)

// PrintNextSteps renders list of [NextSteps] messages and writes them
// to out. It is a no-op if messages is empty.
func PrintNextSteps(out io.Writer, messages []string) {
	if len(messages) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out, aec.Bold.Apply("\nWhat's next:"))
	for _, n := range messages {
		_, _ = fmt.Fprintln(out, "   ", n)
	}
}
