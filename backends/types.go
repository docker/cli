package backends

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

const (
	// ContextTypeECS is Amazon ECS context type
	ContextTypeECS = "ecs"
	// ContextTypeACI is MS Azure Container Instances context type
	ContextTypeACI = "aci"
	// ContextTypeLocal is context type for local engine and compose implemetation in particular
	ContextTypeLocal = "local"
)

// RunBackendCLI replace the current process with dedicated CLI for this context type
func RunBackendCLI(contextType string) error {
	backend, err := GetBackend(contextType)
	if err != nil {
		return fmt.Errorf("unsupported context type: %q", contextType)
	}
	if _, err := os.Stat(backend.Path); os.IsNotExist(err) {
		// TODO we could be more restrictive about supported types to prevent abuses using ContextType enum values
		return fmt.Errorf("unsupported context type %q", contextType)
	}
	delegate(backend.Path) // will exit
	return nil
}

func delegate(execBinary string) {
	cmd := exec.Command(execBinary, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	signals := make(chan os.Signal, 1)
	childExit := make(chan bool)
	signal.Notify(signals) // catch all signals
	go func() {
		for {
			select {
			case sig := <-signals:
				if cmd.Process == nil {
					continue // can happen if receiving signal before the process is actually started
				}
				// nolint errcheck
				cmd.Process.Signal(sig)
			case <-childExit:
				return
			}
		}
	}()

	err := cmd.Run()
	childExit <- true
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			os.Exit(exiterr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
