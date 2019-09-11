package engine

import (
	clitypes "github.com/docker/cli/types"
)

type extendedEngineInitOptions struct {
	clitypes.EngineInitOptions
	sockPath string
}
