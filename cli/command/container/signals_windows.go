package container // import "docker.com/cli/v28/cli/command/container"

import "os"

func isRuntimeSig(_ os.Signal) bool {
	return false
}
