package credentials

import (
	"github.com/docker/docker-credential-helpers/pass"
)

func defaultCredentialsStore() string {
	passStore := pass.Pass{}
	if passStore.CheckInitialized() {
		return "pass"
	}

	return "secretservice"
}
