package completion

import "context"

func NewShellCompletionSetup(homeDirectory string, rootCobraCmd generateCompletions, opts ...NewShellCompletionOptsFunc) (ShellCompletionSetup, error) {
	return newWindowsShellSetup(homeDirectory, rootCobraCmd, opts...)
}

type windowsShellSetup struct {
	common
	powershellCompletionDir string
}

var _ ShellCompletionSetup = (*windowsShellSetup)(nil)

func newWindowsShellSetup(homeDirectory string, rootCobraCmd generateCompletions, opts ...NewShellCompletionOptsFunc) (*windowsShellSetup, error) {
	return nil, nil
}

func (w *windowsShellSetup) GetCompletionScript(ctx context.Context) ([]byte, error) {
	return nil, nil
}

func (w *windowsShellSetup) GetCompletionDir(ctx context.Context) string {
	return ""
}

func (w *windowsShellSetup) InstallCompletions(ctx context.Context) error {
	return nil
}

func (w *windowsShellSetup) GetShell() supportedCompletionShell {
	return ""
}

func (w *windowsShellSetup) GetManualInstructions(ctx context.Context) string {
	return ""
}

func (w *windowsShellSetup) InstallStatus(ctx context.Context) (*ShellCompletionInstallStatus, error) {
	return nil, nil
}
