package completion

import (
	_ "embed" // needed to make embed work
	"errors"
)

var (
	//go:embed bash/docker
	completionBash string

	//go:embed fish/docker.fish
	completionFish string

	//go:embed zsh/_docker
	completionZsh string

	completions = map[string]string{
		"bash": completionBash,
		"fish": completionFish,
		"zsh":  completionZsh,
	}
)

// Get returns the completion script for the given shell (bash, fish, or zsh).
func Get(shell string) (string, error) {
	cs, ok := completions[shell]
	if !ok {
		return "", errors.New("no completion available for: " + shell)
	}
	return cs, nil
}
