package credentials

import (
	"github.com/docker/docker-credential-helpers/pass"
)

func defaultCredentialsStore() string {
	if pass.IsPresent() {
		return "pass"
	}

	return "secretservice"
}
