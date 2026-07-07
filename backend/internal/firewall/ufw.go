package firewall

import (
	"context"
	"regexp"
	"strings"
	"time"

	"firewall-manager/backend/internal/config"
)

type UFWService struct {
	base BaseService
	cfg  config.FirewallConfig
}

func NewUFWService(base BaseService, cfg config.FirewallConfig) *UFWService {
	return &UFWService{base: base, cfg: cfg}
}

func (s *UFWService) LoadState(ctx context.Context) (State, error) {
	result, err := s.base.Runner.Run(ctx, s.cfg.UFWPath, "status", "verbose")
	if err != nil {
		return State{}, Error{Code: "FIREWALL_STATE_LOAD_FAILED", Message: result.Stderr}
	}
	return State{
		OSType:                "ubuntu",
		Backend:               "ufw",
		ServiceEnabled:        systemctlBool(ctx, s.base.Runner, s.cfg.SystemctlPath, "is-enabled", "ufw"),
		ServiceRunning:        strings.Contains(strings.ToLower(result.Stdout), "status: active"),
		DefaultIncomingPolicy: ParseUFWDefaultPolicy(result.Stdout),
		OpenPorts:             ParseUFWPorts(result.Stdout),
		LoadedAt:              time.Now().UTC(),
	}, nil
}

func (s *UFWService) OpenPort(ctx context.Context, request PortChangeRequest) (State, error) {
	req, err := ValidatePortChange(request)
	if err != nil {
		return State{}, err
	}
	specs, _ := ParsePortExpression(req.Port)
	for _, spec := range specs {
		if _, err := s.base.Runner.Run(ctx, s.cfg.UFWPath, "allow", ufwPortArg(spec, req.Protocol)); err != nil {
			return State{}, Error{Code: "PORT_OPEN_FAILED", Message: err.Error()}
		}
	}
	return s.LoadState(ctx)
}

func (s *UFWService) ClosePort(ctx context.Context, request PortChangeRequest) (State, error) {
	req, err := ValidatePortChange(request)
	if err != nil {
		return State{}, err
	}
	specs, _ := ParsePortExpression(req.Port)
	for _, spec := range specs {
		if _, err := s.base.Runner.Run(ctx, s.cfg.UFWPath, "delete", "allow", ufwPortArg(spec, req.Protocol)); err != nil {
			return State{}, Error{Code: "PORT_CLOSE_FAILED", Message: err.Error()}
		}
	}
	return s.LoadState(ctx)
}

func ParseUFWDefaultPolicy(output string) string {
	re := regexp.MustCompile(`(?i)Default:\s+(allow|deny|reject)\s+\(incoming\)`) // UFW verbose format.
	match := re.FindStringSubmatch(output)
	if len(match) == 2 {
		return strings.ToLower(match[1])
	}
	return "unknown"
}

func ParseUFWPorts(output string) []PortRule {
	ports := []PortRule{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(strings.ToLower(line), "status:") || strings.HasPrefix(strings.ToLower(line), "logging:") || strings.HasPrefix(strings.ToLower(line), "default:") || strings.HasPrefix(strings.ToLower(line), "new profiles:") || strings.HasPrefix(strings.ToLower(line), "to ") || strings.HasPrefix(line, "--") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.ToUpper(fields[1]) != "ALLOW" {
			continue
		}
		if strings.Contains(fields[0], "(") {
			continue
		}
		if rule, ok := parsePortToken(fields[0]); ok {
			rule.Source = "Any"
			if len(fields) >= 3 {
				rule.Source = strings.Join(fields[2:], " ")
			}
			ports = append(ports, rule)
		}
	}
	return sortPorts(ports)
}
