//go:build unix

package completion

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

type unixShellSetup struct {
	zshrc             string
	zshCompletionDir  string
	fishCompletionDir string
	bashCompletionDir string
	hasOhMyZsh        bool
	common
}

var (
	ErrCompletionNotInstalled    = errors.New("completion not installed")
	ErrCompletionOutdated        = errors.New("completion file is outdated")
	ErrZshrcCouldNotWrite        = errors.New("could not write to .zshrc file since it may not exist or have the necessary permissions")
	ErrCompletionDirectoryCreate = errors.New("could not create the completions directory")
	ErrCompletionFileWrite       = errors.New("could not write to the completions file")
	ErrCompletionGenerated       = errors.New("could not generate completions")
	ErrZshFpathNotFound          = errors.New("completions file not found in the FPATH environment variable")
)

var _ ShellCompletionSetup = &unixShellSetup{}

func hasCompletionInFpath(zshrc, completionDir string) (bool, error) {
	// check the FPATH environment variable first which contains a string of directories
	if fpathEnv := os.Getenv("FPATH"); fpathEnv != "" && strings.Contains(fpathEnv, completionDir) {
		return true, nil
	}

	if _, err := os.Stat(zshrc); err != nil {
		return false, fmt.Errorf("unable to edit %s since it does not exist. Setup your zsh completions manually or create the .zshrc file inside of your home directory and try again", zshrc)
	}

	// This should error if it does not exist.
	zshrcContent, err := os.ReadFile(zshrc)
	if err != nil {
		return false, fmt.Errorf("unable to edit %s. Make sure that your .zshrc file is set up correctly before continuing", zshrc)
	}

	fpath := fmt.Sprintf("fpath=(%s $fpath)", completionDir)
	if strings.Contains(string(zshrcContent), fpath) {
		return true, nil
	}

	return false, nil
}

func NewShellCompletionSetup(homeDirectory string, generateCompletions generateCompletions, opts ...NewShellCompletionOptsFunc) (ShellCompletionSetup, error) {
	return newUnixShellSetup(homeDirectory, generateCompletions, opts...)
}

func newUnixShellSetup(homeDirectory string, generateCompletions generateCompletions, opts ...NewShellCompletionOptsFunc) (*unixShellSetup, error) {
	shell, shellRawString, err := shellFromEnv()
	if err != nil {
		return nil, err
	}

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

	c := &common{
		homeDirectory:         homeDirectory,
		command:               generateCompletions,
		currentShell:          shell,
		currentShellRawString: shellRawString,
	}

	for _, opt := range opts {
		opt(c)
	}

	u := &unixShellSetup{
		zshrc:             zshrcFile,
		zshCompletionDir:  zshCompletionDir,
		fishCompletionDir: filepath.Join(homeDirectory, fishCompletionDir),
		bashCompletionDir: filepath.Join(homeDirectory, bashCompletionDir),
		hasOhMyZsh:        hasOhMyZsh,
		common:            *c,
	}

	return u, nil
}

func (u *unixShellSetup) GetCompletionScript(ctx context.Context) ([]byte, error) {
	var err error
	var buff bytes.Buffer

	switch u.currentShell {
	case zsh:
		err = u.command.GenZshCompletion(&buff)
	case bash:
		err = u.command.GenBashCompletionV2(&buff, true)
	case fish:
		err = u.command.GenFishCompletion(&buff, true)
	default:
		return nil, ErrShellUnsupported
	}

	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func (u *unixShellSetup) GetCompletionDir(ctx context.Context) string {
	switch u.currentShell {
	case zsh:
		return u.zshCompletionDir
	case fish:
		return u.fishCompletionDir
	case bash:
		return u.bashCompletionDir
	}
	return ""
}

func (u *unixShellSetup) GetManualInstructions(ctx context.Context) string {
	completionDir := u.GetCompletionDir(ctx)
	completionsFile := filepath.Join(completionDir, u.currentShell.FileName())

	instructions := fmt.Sprintf("\tmkdir -p %s\n\tdocker completion %s > %s", completionDir, u.currentShell, completionsFile)

	if u.currentShell == zsh && !u.hasOhMyZsh {
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

func (u *unixShellSetup) InstallCompletions(ctx context.Context) error {
	completionDir := u.GetCompletionDir(ctx)

	if err := os.MkdirAll(completionDir, filePerm); err != nil {
		return err
	}

	completionFile := filepath.Join(completionDir, u.currentShell.FileName())

	_ = os.Remove(completionFile)

	completions, err := u.GetCompletionScript(ctx)
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
	if u.currentShell == zsh && !u.hasOhMyZsh {

		// This should error if it does not exist.
		zshrcContent, err := os.ReadFile(u.zshrc)
		if err != nil {
			// TODO: what should we do here? The error message might not be too helpful.
			return fmt.Errorf("could not open %s. Please ensure that your .zshrc file is set up correctly before continuing", u.zshrc)
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

func (u *unixShellSetup) GetShell() supportedCompletionShell {
	return u.currentShell
}

func (u *unixShellSetup) InstallStatus(ctx context.Context) (*ShellCompletionInstallStatus, error) {
	installStatus := &ShellCompletionInstallStatus{
		Shell:  u.currentShellRawString,
		Status: StatusNotInstalled,
	}

	ok, err := u.currentShell.Supported()
	if !ok {
		installStatus.Status = StatusUnsupported
	}

	if err != nil {
		installStatus.Reason = err.Error()
		return installStatus, nil
	}

	completionDir := u.GetCompletionDir(ctx)
	completionFile := filepath.Join(completionDir, u.currentShell.FileName())
	installStatus.CompletionPath = completionFile

	if _, err := os.Stat(completionFile); err != nil {
		installStatus.Reason = ErrCompletionNotInstalled.Error()
		return installStatus, nil
	}

	completionContent, err := os.ReadFile(completionFile)
	if err != nil {
		return installStatus, fmt.Errorf("could not open existing completion file: %s", err.Error())
	}

	completionGenerated, err := u.GetCompletionScript(ctx)
	if err != nil {
		return installStatus, fmt.Errorf("could not generate cli completions: %s", err)
	}

	if !strings.EqualFold(string(completionContent), string(completionGenerated)) {
		installStatus.Status = StatusOutdated
		installStatus.Reason = ErrCompletionOutdated.Error()
		return installStatus, nil
	}

	if u.currentShell == zsh && !u.hasOhMyZsh {
		hasFpath, err := hasCompletionInFpath(u.zshrc, completionDir)
		if err != nil || !hasFpath {
			installStatus.Reason = ErrZshFpathNotFound.Error()
			return installStatus, nil
		}
		f, err := os.Stat(u.zshrc)
		if err != nil {
			installStatus.Reason = ErrZshrcCouldNotWrite.Error()
			return installStatus, nil
		}
		if f.Mode().Perm() < 0o600 {
			installStatus.Reason = ErrZshrcCouldNotWrite.Error()
			return installStatus, nil
		}
	}

	installStatus.Status = StatusInstalled
	installStatus.Reason = fmt.Sprintf("Shell completion already installed for %s.", u.currentShell)

	return installStatus, nil
}
