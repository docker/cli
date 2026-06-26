package hooks

import "io"

const (
	whatsNext = "\n\033[1mWhat's next:\033[0m\n"
	indent    = "    "
)

// PrintNextSteps renders list of [NextSteps] messages and writes them
// to out. It is a no-op if messages is empty.
func PrintNextSteps(out io.Writer, messages []string) {
	if len(messages) == 0 {
		return
	}

	_, _ = io.WriteString(out, whatsNext)
	for _, msg := range messages {
		_, _ = io.WriteString(out, indent)
		_, _ = io.WriteString(out, msg)
		_, _ = io.WriteString(out, "\n")
	}
}
