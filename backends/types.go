package backends

import (
	"fmt"
	"os"
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
