package tui

type Str struct {
	// Fancy is the fancy string representation of the string.
	Fancy string

	// Plain is the plain string representation of the string.
	Plain string
}

func (p Str) String(isTerminal bool) string {
	if isTerminal {
		return p.Fancy
	}
	return p.Plain
}
