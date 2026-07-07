package firewall

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"firewall-manager/backend/internal/command"
)

type BaseService struct {
	Runner command.Runner
}

type PortSpec struct {
	Value string
	Start int
	End   int
}

func (r *PortChangeRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		Port     any    `json:"port"`
		Protocol string `json:"protocol"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch value := raw.Port.(type) {
	case string:
		r.Port = value
	case float64:
		if value == float64(int(value)) {
			r.Port = strconv.Itoa(int(value))
		}
	}
	r.Protocol = raw.Protocol
	return nil
}

func ValidatePortChange(request PortChangeRequest) (PortChangeRequest, error) {
	request.Protocol = strings.ToLower(strings.TrimSpace(request.Protocol))
	if request.Protocol != "tcp" && request.Protocol != "udp" {
		return request, Error{Code: "PROTOCOL_INVALID", Message: "protocol must be tcp or udp"}
	}
	specs, err := ParsePortExpression(request.Port)
	if err != nil {
		return request, err
	}
	values := make([]string, 0, len(specs))
	for _, spec := range specs {
		values = append(values, spec.Value)
	}
	request.Port = strings.Join(values, ",")
	return request, nil
}

func ParsePortExpression(expression string) ([]PortSpec, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil, Error{Code: "PORT_INVALID", Message: "port expression is empty"}
	}
	parts := strings.Split(expression, ",")
	specs := make([]PortSpec, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, Error{Code: "PORT_INVALID", Message: "empty port item"}
		}
		part = strings.ReplaceAll(part, ":", "-")
		bounds := strings.Split(part, "-")
		if len(bounds) > 2 {
			return nil, Error{Code: "PORT_INVALID", Message: "invalid port range"}
		}
		start, err := parsePortNumber(bounds[0])
		if err != nil {
			return nil, err
		}
		end := start
		if len(bounds) == 2 {
			end, err = parsePortNumber(bounds[1])
			if err != nil {
				return nil, err
			}
			if start > end {
				return nil, Error{Code: "PORT_INVALID", Message: "range start must be <= end"}
			}
		}
		value := strconv.Itoa(start)
		if start != end {
			value = fmt.Sprintf("%d-%d", start, end)
		}
		specs = append(specs, PortSpec{Value: value, Start: start, End: end})
	}
	return specs, nil
}

func parsePortNumber(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, Error{Code: "PORT_INVALID", Message: "empty port"}
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, Error{Code: "PORT_INVALID", Message: "port must be numeric"}
		}
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, Error{Code: "PORT_INVALID", Message: "port must be between 1 and 65535"}
	}
	return port, nil
}

func parsePortToken(token string) (PortRule, bool) {
	parts := strings.Split(token, "/")
	if len(parts) != 2 {
		return PortRule{}, false
	}
	specs, err := ParsePortExpression(parts[0])
	if err != nil || len(specs) != 1 {
		return PortRule{}, false
	}
	protocol := strings.ToLower(strings.TrimSpace(parts[1]))
	if protocol != "tcp" && protocol != "udp" {
		return PortRule{}, false
	}
	return PortRule{Port: specs[0].Value, Protocol: protocol, Source: "Any"}, true
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
	return fmt.Sprintf("%s/%s", request.Port, request.Protocol)
}

func ufwPortArg(spec PortSpec, protocol string) string {
	value := spec.Value
	if spec.Start != spec.End {
		value = strings.ReplaceAll(value, "-", ":")
	}
	return fmt.Sprintf("%s/%s", value, protocol)
}

func firewalldPortArg(spec PortSpec, protocol string) string {
	return fmt.Sprintf("%s/%s", spec.Value, protocol)
}
