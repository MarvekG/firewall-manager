package firewall

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"firewall-manager/backend/internal/config"
)

func TestNewServiceExplicitBackends(t *testing.T) {
	cfg := config.FirewallConfig{Backend: "mock", CommandTimeout: time.Second}
	service, err := NewService(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("mock service failed: %v", err)
	}
	if _, ok := service.(*MockService); !ok {
		t.Fatalf("expected mock service, got %T", service)
	}

	cfg.Backend = "ufw"
	service, err = NewService(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("ufw service failed: %v", err)
	}
	if _, ok := service.(*UbuntuService); !ok {
		t.Fatalf("expected UbuntuService, got %T", service)
	}

	cfg.Backend = "firewalld"
	service, err = NewService(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("firewalld service failed: %v", err)
	}
	if _, ok := service.(*CentOSService); !ok {
		t.Fatalf("expected CentOSService, got %T", service)
	}
}

func TestNewServiceAutoDetectsOSRelease(t *testing.T) {
	dir := t.TempDir()
	osRelease := filepath.Join(dir, "os-release")
	if err := os.WriteFile(osRelease, []byte("ID=ubuntu\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	service, err := NewService(context.Background(), config.FirewallConfig{Backend: "auto", OSReleasePath: osRelease, UFWPath: os.Args[0], CommandTimeout: time.Second}, nil)
	if err != nil {
		t.Fatalf("auto service failed: %v", err)
	}
	if _, ok := service.(*UbuntuService); !ok {
		t.Fatalf("expected UbuntuService, got %T", service)
	}
}

func TestNewServiceUnsupportedOS(t *testing.T) {
	dir := t.TempDir()
	osRelease := filepath.Join(dir, "os-release")
	if err := os.WriteFile(osRelease, []byte("ID=unsupported\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := NewService(context.Background(), config.FirewallConfig{Backend: "auto", OSReleasePath: osRelease, CommandTimeout: time.Second}, nil)
	if err == nil {
		t.Fatalf("expected unsupported OS error")
	}
	if fwErr, ok := err.(Error); !ok || fwErr.Code != "UNSUPPORTED_OS" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
