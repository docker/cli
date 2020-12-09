package context

import (
	"fmt"
	"os"
	"syscall"
)

const (
	// ContextTypeECS is Amazon ECS context type
	ContextTypeECS = "ecs"

	// ContextTypeACI is MS Azure Container Instances context type
	ContextTypeACI = "aci"
)

// RunContextCLI replace the current process with dedicated CLI for this context type
func RunContextCLI(context string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	x := executable + "-" + context
	if _, err := os.Stat(x); os.IsNotExist(err) {
		// TODO we could be more restrictive about supported types to prevent abuses using ContextType enum values
		return fmt.Errorf("unsupported context type %q", context)
	}
	return syscall.Exec(x, os.Args[1:], os.Environ())
}
