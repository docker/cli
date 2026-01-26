package debug

import (
	"os"
	"testing"

	"github.com/containerd/log"
)

func TestEnable(t *testing.T) {
	defer func() {
		_ = log.SetLevel("info")
	}()
	t.Setenv("DEBUG", "")
	Enable()
	if os.Getenv("DEBUG") != "1" {
		t.Fatalf("expected DEBUG=1, got %s\n", os.Getenv("DEBUG"))
	}
	if log.GetLevel() != log.DebugLevel {
		t.Fatalf("expected log level %v, got %v\n", log.DebugLevel, log.GetLevel())
	}
}

func TestDisable(t *testing.T) {
	t.Setenv("DEBUG", "1")
	Disable()
	if os.Getenv("DEBUG") != "" {
		t.Fatalf("expected DEBUG=\"\", got %s\n", os.Getenv("DEBUG"))
	}
	if log.GetLevel() != log.InfoLevel {
		t.Fatalf("expected log level %v, got %v\n", log.InfoLevel, log.GetLevel())
	}
}

func TestEnabled(t *testing.T) {
	t.Setenv("DEBUG", "")
	Enable()
	if !IsEnabled() {
		t.Fatal("expected debug enabled, got false")
	}
	Disable()
	if IsEnabled() {
		t.Fatal("expected debug disabled, got true")
	}
}
