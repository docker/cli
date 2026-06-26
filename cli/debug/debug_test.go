package debug

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestEnable(t *testing.T) {
	defer func() {
		logrus.SetLevel(logrus.InfoLevel)
	}()
	t.Setenv("DEBUG", "")
	Enable()
	if os.Getenv("DEBUG") != "1" {
		t.Fatalf("expected DEBUG=1, got %s\n", os.Getenv("DEBUG"))
	}
	if logrus.GetLevel() != logrus.DebugLevel {
		t.Fatalf("expected log level %v, got %v\n", logrus.DebugLevel, logrus.GetLevel())
	}
}

func TestDisable(t *testing.T) {
	t.Setenv("DEBUG", "1")
	Disable()
	if os.Getenv("DEBUG") != "" {
		t.Fatalf("expected DEBUG=\"\", got %s\n", os.Getenv("DEBUG"))
	}
	if logrus.GetLevel() != logrus.InfoLevel {
		t.Fatalf("expected log level %v, got %v\n", logrus.InfoLevel, logrus.GetLevel())
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
