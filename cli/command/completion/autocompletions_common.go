package completion

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
)

type ShellCompletionSetup interface {
	// Generate completions for the Docker CLI based on the provided shell.
	GetCompletionScript(ctx context.Context) ([]byte, error)
	// Set up completions for the Docker CLI for the provided shell.
	//
	// For zsh completions, this function should also configure the user's
	// .zshrc file to load the completions correctly.
	// Please see https://zsh.sourceforge.io/Doc/Release/Completion-System.html
	// for more information.
	InstallCompletions(ctx context.Context) error
	// Check if the shell completion is already installed.
	InstallStatus(ctx context.Context) (*ShellCompletionInstallStatus, error)
	// Get the completion directory for the provided shell.
	GetCompletionDir(ctx context.Context) string
	// Get the manual instructions for the provided shell.
	GetManualInstructions(ctx context.Context) string
	// Get he current supported shell
	GetShell() supportedCompletionShell
}

type completionStatus string

const (
	StatusInstalled    completionStatus = "INSTALLED"
	StatusNotInstalled completionStatus = "NOT_INSTALLED"
	StatusUnsupported  completionStatus = "UNSUPPORTED"
	StatusOutdated     completionStatus = "OUTDATED"
)

var (
	ErrShellEnvNotSet   = errors.New("SHELL environment variable not set")
	ErrShellUnsupported = errors.New("unsupported shell")
)

type supportedCompletionShell string

const (
	bash       supportedCompletionShell = "bash"
	fish       supportedCompletionShell = "fish"
	zsh        supportedCompletionShell = "zsh"
	powershell supportedCompletionShell = "powershell"
)

func (s supportedCompletionShell) FileName() string {
	switch s {
	case zsh:
		return "_docker"
	case bash:
		return "docker"
	case fish:
		return "docker.fish"
	}
	return ""
}

func (s supportedCompletionShell) Supported() (bool, error) {
	switch s {
	case zsh, bash, fish:
		return true, nil
	}
	return false, ErrShellUnsupported
}

type ShellCompletionInstallStatus struct {
	Status         completionStatus
	Shell          string
	CompletionPath string
	Reason         string
}

type common struct {
	homeDirectory         string
	command               generateCompletions
	currentShell          supportedCompletionShell
	currentShellRawString string
}

type generateCompletions interface {
	GenBashCompletionV2(w io.Writer, includeDesc bool) error
	GenZshCompletion(w io.Writer) error
	GenFishCompletion(w io.Writer, includeDesc bool) error
}

// shellFromEnv returns the shell type and its name.
func shellFromEnv() (supportedCompletionShell, string, error) {
	currentShell := os.Getenv("SHELL")

	if len(currentShell) == 0 {
		return "", "", ErrShellEnvNotSet
	}

	t := strings.Split(currentShell, "/")
	shellName := t[len(t)-1]

	if ok, err := supportedCompletionShell(shellName).Supported(); !ok {
		return "", shellName, err
	}

	return supportedCompletionShell(shellName), shellName, nil
}

type NewShellCompletionOptsFunc func(*common)

func WithShellOverride(shell string) NewShellCompletionOptsFunc {
	return func(u *common) {
		u.currentShell = supportedCompletionShell(shell)
		u.currentShellRawString = shell
	}
}
