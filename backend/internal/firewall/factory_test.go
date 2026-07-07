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

func TestNewServiceAutoDetectsFirewallCmdFirst(t *testing.T) {
	dir := t.TempDir()
	firewallCmd := filepath.Join(dir, "firewall-cmd")
	ufw := filepath.Join(dir, "ufw")
	writeExecutable(t, firewallCmd)
	writeExecutable(t, ufw)

	service, err := NewService(context.Background(), config.FirewallConfig{Backend: "auto", FirewallCmdPath: firewallCmd, UFWPath: ufw, CommandTimeout: time.Second}, nil)
	if err != nil {
		t.Fatalf("auto service failed: %v", err)
	}
	if _, ok := service.(*CentOSService); !ok {
		t.Fatalf("expected CentOSService, got %T", service)
	}
}

func TestNewServiceAutoDetectsUFWWhenFirewallCmdMissing(t *testing.T) {
	dir := t.TempDir()
	ufw := filepath.Join(dir, "ufw")
	writeExecutable(t, ufw)

	service, err := NewService(context.Background(), config.FirewallConfig{Backend: "auto", FirewallCmdPath: filepath.Join(dir, "missing-firewall-cmd"), UFWPath: ufw, CommandTimeout: time.Second}, nil)
	if err != nil {
		t.Fatalf("auto service failed: %v", err)
	}
	if _, ok := service.(*UbuntuService); !ok {
		t.Fatalf("expected UbuntuService, got %T", service)
	}
}

func TestNewServiceUnsupportedOS(t *testing.T) {
	dir := t.TempDir()

	_, err := NewService(context.Background(), config.FirewallConfig{Backend: "auto", FirewallCmdPath: filepath.Join(dir, "missing-firewall-cmd"), UFWPath: filepath.Join(dir, "missing-ufw"), CommandTimeout: time.Second}, nil)
	if err == nil {
		t.Fatalf("expected unsupported backend error")
	}
	if fwErr, ok := err.(Error); !ok || fwErr.Code != "UNSUPPORTED_OS" {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
}
