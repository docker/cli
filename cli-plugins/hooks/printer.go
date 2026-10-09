package hooks

import "io"

const (
	whatsNext = "\n\033[1mWhat's next:\033[0m\n"
	indent    = "    "
)

// EvaluatedMessage contains the lines of a hook response after template evaluation.
type EvaluatedMessage struct {
	Type  ResponseType
	Lines []string
}

// PrintMessages groups messages by type and writes them to out. Generic messages
// appear first, without a header or indentation, followed by next steps under a
// shared "What's next:" header. Order within each type is preserved.
// Messages with no lines or an unknown type produce no output.
func PrintMessages(out io.Writer, messages []EvaluatedMessage) {
	grouped := make(map[ResponseType][]string)
	for _, message := range messages {
		grouped[message.Type] = append(grouped[message.Type], message.Lines...)
	}

	for _, responseType := range []ResponseType{GenericMessage, NextSteps} {
		lines := grouped[responseType]
		if len(lines) == 0 {
			continue
		}

		header, prefix := "\n", ""
		if responseType == NextSteps {
			header, prefix = whatsNext, indent
		}

		_, _ = io.WriteString(out, header)
		for _, line := range lines {
			_, _ = io.WriteString(out, prefix)
			_, _ = io.WriteString(out, line)
			_, _ = io.WriteString(out, "\n")
		}
	}
}

// PrintNextSteps renders list of [NextSteps] messages and writes them
// to out. It is a no-op if messages is empty.
//
// Deprecated: use [PrintMessages] instead.
func PrintNextSteps(out io.Writer, messages []string) {
	PrintMessages(out, []EvaluatedMessage{{Type: NextSteps, Lines: messages}})
}
