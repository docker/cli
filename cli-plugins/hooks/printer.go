package hooks // import "docker.com/cli/v28/cli-plugins/hooks"

import (
	"fmt"
	"io"

	"github.com/morikuni/aec"
)

func PrintNextSteps(out io.Writer, messages []string) {
	if len(messages) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out, aec.Bold.Apply("\nWhat's next:"))
	for _, n := range messages {
		_, _ = fmt.Fprintln(out, "   ", n)
	}
}
