package tui

import (
	"github.com/morikuni/aec"
)

var (
	ColorTitle     = aec.NewBuilder(aec.DefaultF, aec.Bold).ANSI
	ColorPrimary   = aec.NewBuilder(aec.DefaultF, aec.Bold).ANSI
	ColorSecondary = aec.DefaultF
	ColorTertiary  = aec.NewBuilder(aec.DefaultF, aec.Faint).ANSI
	ColorLink      = aec.NewBuilder(aec.LightCyanF, aec.Underline).ANSI
	ColorWarning   = aec.LightYellowF
	ColorFlag      = aec.NewBuilder(aec.Bold).ANSI
	ColorNone      = aec.ANSI(noColor{})
)

type noColor struct{}

func (a noColor) With(_ ...aec.ANSI) aec.ANSI {
	return a
}

func (a noColor) Apply(s string) string {
	return s
}

func (a noColor) String() string {
	return ""
}
