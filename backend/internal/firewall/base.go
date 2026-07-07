package firewall

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"firewall-manager/backend/internal/command"
)

type BaseService struct {
	Runner command.Runner
}

func ValidatePortChange(request PortChangeRequest) (PortChangeRequest, error) {
	if request.Port < 1 || request.Port > 65535 {
		return request, Error{Code: "PORT_INVALID", Message: "port must be between 1 and 65535"}
	}
	request.Protocol = strings.ToLower(strings.TrimSpace(request.Protocol))
	if request.Protocol != "tcp" && request.Protocol != "udp" {
		return request, Error{Code: "PROTOCOL_INVALID", Message: "protocol must be tcp or udp"}
	}
	return request, nil
}

func parsePortToken(token string) (PortRule, bool) {
	parts := strings.Split(token, "/")
	if len(parts) != 2 {
		return PortRule{}, false
	}
	if strings.Contains(parts[0], "-") {
		return PortRule{}, false
	}
	port, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || port < 1 || port > 65535 {
		return PortRule{}, false
	}
	protocol := strings.ToLower(strings.TrimSpace(parts[1]))
	if protocol != "tcp" && protocol != "udp" {
		return PortRule{}, false
	}
	return PortRule{Port: port, Protocol: protocol, Source: "Any"}, true
}

func systemctlBool(ctx context.Context, runner command.Runner, systemctlPath, action, unit string) bool {
	result, err := runner.Run(ctx, systemctlPath, action, unit)
	return err == nil && strings.TrimSpace(result.Stdout) == actionValue(action)
}

func actionValue(action string) string {
	switch action {
	case "is-enabled":
		return "enabled"
	case "is-active":
		return "active"
	default:
		return ""
	}
}

func portArg(request PortChangeRequest) string {
	return fmt.Sprintf("%d/%s", request.Port, request.Protocol)
}
