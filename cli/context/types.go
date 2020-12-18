package context

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
)

const (
	// ContextTypeECS is Amazon ECS context type
	ContextTypeECS = "ecs"
	// ContextTypeACI is MS Azure Container Instances context type
	ContextTypeACI = "aci"

	//TODO move in a separate file and have a windows version
	backendFolder = "/usr/local/lib/docker/cli-backends"
)

// RunContextCLI replace the current process with dedicated CLI for this context type
func RunContextCLI(context string) error {
	if context != ContextTypeACI && context != ContextTypeECS {
		return fmt.Errorf("unsupported context type: %q", context)
	}
	x := filepath.Join(backendFolder, "compose-cli")
	if _, err := os.Stat(x); os.IsNotExist(err) {
		// TODO we could be more restrictive about supported types to prevent abuses using ContextType enum values
		return fmt.Errorf("unsupported context type %q", context)
	}
	delegate(x) // will exit
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
