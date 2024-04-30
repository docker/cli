package completion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type shellCompletionSetup interface {
	// Generate completions for the Docker CLI based on the provided shell.
	DockerCompletion(ctx context.Context, shell supportedCompletionShell) ([]byte, error)
	// Set up completions for the Docker CLI for the provided shell.
	//
	// For zsh completions, this function should also configure the user's
	// .zshrc file to load the completions correctly.
	// Please see https://zsh.sourceforge.io/Doc/Release/Completion-System.html
	// for more information.
	InstallCompletions(ctx context.Context, shell supportedCompletionShell) error
	// Get the completion directory for the provided shell.
	GetCompletionDir(shell supportedCompletionShell) string
	// Get the manual instructions for the provided shell.
	GetManualInstructions(shell supportedCompletionShell) string
}

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

const (
	zshrc             = ".zshrc"
	zshCompletionDir  = ".docker/completions"
	fishCompletionDir = ".config/fish/completions"
	bashCompletionDir = ".local/share/bash-completion/completions"
)

// TODO: file permissions are difficult.
// Wondering if this should be 0644 or 0750
// From stackoverflow most mention a sane default for the home directory
// is 0755/0751.
const filePerm = 0755

type common struct {
	command         func(ctx context.Context, name string, arg ...string) *exec.Cmd
	homeDirectory   string
	dockerCliBinary string
}

type unixShellSetup struct {
	zshrc             string
	zshCompletionDir  string
	fishCompletionDir string
	bashCompletionDir string
	hasOhMyZsh        bool
	common
}

func unixDefaultShell() (supportedCompletionShell, error) {
	currentShell := os.Getenv("SHELL")

	if len(currentShell) == 0 {
		return "", errors.New("SHELL environment variable not set")
	}

	t := strings.Split(currentShell, "/")

	switch t[len(t)-1] {
	case "bash":
		return bash, nil
	case "zsh":
		return zsh, nil
	case "fish":
		return fish, nil
	}

	return "", errors.New("unsupported shell")
}

var _ shellCompletionSetup = &unixShellSetup{}

func NewUnixShellSetup(homeDirectory string, dockerCliBinary string) shellCompletionSetup {
	zshrcFile := filepath.Join(homeDirectory, zshrc)
	// override the default directory if ZDOTDIR is set
	// if this is set, we assume the user has set up their own zshrc
	// and we should append to that file instead
	if zshroot := os.Getenv("ZDOTDIR"); zshroot != "" {
		zshrcFile = filepath.Join(zshroot, zshrc)
	}
	var hasOhMyZsh bool
	zshCompletionDir := filepath.Join(homeDirectory, zshCompletionDir)
	// overide the default zsh completions directory if oh-my-zsh is installed
	if ohmyzsh := os.Getenv("ZSH"); ohmyzsh != "" {
		// ensure that the oh-my-zsh completions directory exists
		if _, err := os.Stat(ohmyzsh); err == nil {
			zshCompletionDir = filepath.Join(ohmyzsh, "completions")
			hasOhMyZsh = true
		}
	}
	return &unixShellSetup{
		zshrc:             zshrcFile,
		zshCompletionDir:  zshCompletionDir,
		fishCompletionDir: filepath.Join(homeDirectory, fishCompletionDir),
		bashCompletionDir: filepath.Join(homeDirectory, bashCompletionDir),
		hasOhMyZsh:        hasOhMyZsh,
		common: common{
			homeDirectory:   homeDirectory,
			dockerCliBinary: dockerCliBinary,
			command:         exec.CommandContext,
		},
	}
}

func (u *unixShellSetup) DockerCompletion(ctx context.Context, shell supportedCompletionShell) ([]byte, error) {
	dockerCmd := u.command(ctx, u.dockerCliBinary, "completion", string(shell))
	out, err := dockerCmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (u *unixShellSetup) GetCompletionDir(shell supportedCompletionShell) string {
	switch shell {
	case zsh:
		return u.zshCompletionDir
	case fish:
		return u.fishCompletionDir
	case bash:
		return u.bashCompletionDir
	}
	return ""
}

func (u *unixShellSetup) GetManualInstructions(shell supportedCompletionShell) string {
	completionDir := u.GetCompletionDir(shell)
	completionsFile := filepath.Join(completionDir, shell.FileName())

	instructions := fmt.Sprintf(`mkdir -p %s && docker completion %s > %s`, completionDir, shell, completionsFile)

	if shell == zsh && !u.hasOhMyZsh {
		instructions += "\n"
		instructions += fmt.Sprintf("cat <<EOT >> %s\n"+
			"# The following lines have been added by Docker to enable Docker CLI completions.\n"+
			"fpath=(%s $fpath)\n"+
			"autoload -Uz compinit\n"+
			"compinit\n"+
			"EOT\n"+
			"# End of Docker Completions", u.zshrc, completionsFile)
	}

	return instructions
}

func (u *unixShellSetup) InstallCompletions(ctx context.Context, shell supportedCompletionShell) error {
	completionDir := u.GetCompletionDir(shell)

	if err := os.MkdirAll(completionDir, filePerm); err != nil {
		return err
	}

	completionFile := filepath.Join(completionDir, shell.FileName())

	_ = os.Remove(completionFile)

	completions, err := u.DockerCompletion(ctx, shell)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(completionFile, os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = f.Write(completions); err != nil {
		return err
	}

	// only configure fpath for zsh if oh-my-zsh is not installed
	if shell == zsh && !u.hasOhMyZsh {

		// This should error if it does not exist.
		zshrcContent, err := os.ReadFile(u.zshrc)
		if err != nil {
			// TODO: what should we do here? The error message might not be too helpful.
			return fmt.Errorf("could not open %s. Please ensure that your .zshrc file is set up correctly before continuing.", u.zshrc)
		}

		fpath := fmt.Sprintf("fpath=(%s $fpath)", completionDir)
		autoload := "autoload -Uz compinit"
		compinit := "compinit"

		// if fpath is already in the .zshrc file, we don't need to add it again
		if strings.Contains(string(zshrcContent), fpath) {
			return nil
		}

		// Only append to .zshrc when it exists.
		f, err = os.OpenFile(u.zshrc, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			return err
		}
		defer f.Close()

		zshrcFpath := fmt.Sprintf("# The following lines have been added by Docker Desktop to enable Docker CLI completions.\n"+
			"%s\n"+
			"%s\n"+
			"%s\n"+
			"# End of Docker CLI completions\n", fpath, autoload, compinit)

		_, err = f.WriteString(zshrcFpath)
		if err != nil {
			return err
		}
	}

	return nil
}
