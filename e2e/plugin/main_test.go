package plugin

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/cli/v28/internal/test/environment"
)

func TestMain(m *testing.M) {
	if err := environment.Setup(); err != nil {
		fmt.Println(err.Error())
		os.Exit(3)
	}
	os.Exit(m.Run())
}
