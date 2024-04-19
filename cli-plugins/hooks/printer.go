package hooks

import (
	"fmt"
	"io"

	"github.com/morikuni/aec"
)

func PrintNextSteps(out io.Writer, messages []string) {
	if len(messages) == 0 {
		return
	}
	fmt.Fprintln(out, aec.Bold.Apply("\nWhat's next:"))
	for _, n := range messages {
		_, _ = fmt.Fprintf(out, "    %s\n", n)
	}
}
