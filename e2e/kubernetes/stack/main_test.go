package stack

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/cli/internal/test/environment"
	"github.com/docker/cli/internal/test/environment/kubernetes"
	"github.com/docker/cli/internal/test/environment/kubernetes/compose"
)

const (
	kubeconfigPath = "/tmp/kube.config"
)

func TestMain(m *testing.M) {
	if err := environment.Setup(); err != nil {
		fmt.Println(err.Error())
		os.Exit(3)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := kubernetes.Setup(ctx, os.Stderr, kubeconfigPath); err != nil {
		fmt.Println(err.Error())
		os.Exit(3)
	}
	if err := compose.Setup(os.Stderr, kubeconfigPath, compose.Options{
		Namespace: "docker",
		Tag:       "v0.1.2",
		ReconciliationInterval: 100 * time.Millisecond,
	}); err != nil {
		fmt.Println(err.Error())
		os.Exit(3)
	}
	os.Exit(m.Run())
}
