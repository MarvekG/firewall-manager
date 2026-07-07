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
	case "mock", "development":
		return NewMockService(), nil
	case "ufw", "ubuntu":
		return NewUbuntuService(base, cfg), nil
	case "firewalld", "centos":
		return NewCentOSService(base, cfg), nil
	case "auto", "":
		osID := readOSID(cfg.OSReleasePath)
		if osID == "ubuntu" {
			return NewUbuntuService(base, cfg), nil
		}
		if osID == "centos" || osID == "rhel" || osID == "rocky" || osID == "almalinux" {
			return NewCentOSService(base, cfg), nil
		}
		return nil, Error{Code: "UNSUPPORTED_OS", Message: "unsupported os id: " + osID}
	default:
		return nil, Error{Code: "UNSUPPORTED_OS", Message: "unsupported firewall backend: " + cfg.Backend}
	}
}

func readOSID(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
	}
	return ""
}
