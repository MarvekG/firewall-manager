package firewall

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"firewall-manager/backend/internal/command"
	"firewall-manager/backend/internal/config"
)

func NewService(ctx context.Context, cfg config.FirewallConfig, logger *slog.Logger) (Service, error) {
	runner := command.Runner{Timeout: cfg.CommandTimeout, UseSudo: cfg.UseSudo, Logger: logger}
	base := BaseService{Runner: runner}

	switch strings.ToLower(cfg.Backend) {
	case "ufw", "ubuntu":
		return NewUbuntuService(base, cfg), nil
	case "firewalld", "centos":
		return NewCentOSService(base, cfg), nil
	case "auto", "":
		if commandExists(cfg.FirewallCmdPath) {
			return NewCentOSService(base, cfg), nil
		}
		if commandExists(cfg.UFWPath) {
			return NewUbuntuService(base, cfg), nil
		}
		return nil, Error{Code: "UNSUPPORTED_OS", Message: "no supported firewall command found"}
	default:
		return nil, Error{Code: "UNSUPPORTED_OS", Message: "unsupported firewall backend: " + cfg.Backend}
	}
}

func commandExists(name string) bool {
	if name == "" {
		return false
	}
	if strings.Contains(name, "/") {
		info, err := os.Stat(name)
		return err == nil && !info.IsDir()
	}
	paths := []string{"/usr/sbin/", "/usr/bin/", "/sbin/", "/bin/"}
	for _, path := range paths {
		info, err := os.Stat(path + name)
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}
