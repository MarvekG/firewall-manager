package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("FIREWALL_MANAGER_PORT", "")
	t.Setenv("FIREWALL_MANAGER_HOST", "")
	t.Setenv("FIREWALL_MANAGER_FIREWALL_BACKEND", "")

	cfg := Load()
	if cfg.Server.Port != 10240 {
		t.Fatalf("expected default port 10240, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" || !cfg.Server.IsLoopback() {
		t.Fatalf("unexpected host defaults: %#v", cfg.Server)
	}
	if cfg.Firewall.Backend != "auto" || !cfg.Firewall.UseSudo {
		t.Fatalf("unexpected firewall defaults: %#v", cfg.Firewall)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("FIREWALL_MANAGER_HOST", "0.0.0.0")
	t.Setenv("FIREWALL_MANAGER_PORT", "12345")
	t.Setenv("FIREWALL_MANAGER_TLS_ENABLED", "true")
	t.Setenv("FIREWALL_MANAGER_FIREWALL_BACKEND", "ufw")
	t.Setenv("FIREWALL_MANAGER_USE_SUDO", "false")

	cfg := Load()
	if cfg.Server.Host != "0.0.0.0" || cfg.Server.Port != 12345 || !cfg.Server.TLS.Enabled {
		t.Fatalf("unexpected server config: %#v", cfg.Server)
	}
	if cfg.Firewall.Backend != "ufw" || cfg.Firewall.UseSudo {
		t.Fatalf("unexpected firewall config: %#v", cfg.Firewall)
	}
}
