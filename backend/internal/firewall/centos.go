package firewall

import (
	"context"
	"strings"
	"time"

	"firewall-manager/backend/internal/config"
)

type CentOSService struct {
	base BaseService
	cfg  config.FirewallConfig
}

func NewCentOSService(base BaseService, cfg config.FirewallConfig) *CentOSService {
	return &CentOSService{base: base, cfg: cfg}
}

func (s *CentOSService) LoadState(ctx context.Context) (State, error) {
	zone, err := s.zone(ctx)
	if err != nil {
		return State{}, err
	}
	portsResult, err := s.base.Runner.Run(ctx, s.cfg.FirewallCmdPath, "--zone="+zone, "--list-ports")
	if err != nil {
		return State{}, Error{Code: "FIREWALL_STATE_LOAD_FAILED", Message: portsResult.Stderr}
	}
	return State{
		OSType:                "centos",
		Backend:               "firewalld",
		ServiceEnabled:        systemctlBool(ctx, s.base.Runner, s.cfg.SystemctlPath, "is-enabled", "firewalld"),
		ServiceRunning:        systemctlBool(ctx, s.base.Runner, s.cfg.SystemctlPath, "is-active", "firewalld"),
		DefaultIncomingPolicy: "unknown",
		OpenPorts:             ParseFirewalldPorts(portsResult.Stdout),
		LoadedAt:              time.Now().UTC(),
	}, nil
}

func (s *CentOSService) OpenPort(ctx context.Context, request PortChangeRequest) (State, error) {
	req, err := ValidatePortChange(request)
	if err != nil {
		return State{}, err
	}
	specs, _ := ParsePortExpression(req.Port)
	zone, err := s.zone(ctx)
	if err != nil {
		return State{}, err
	}
	for _, spec := range specs {
		arg := firewalldPortArg(spec, req.Protocol)
		if _, err := s.base.Runner.Run(ctx, s.cfg.FirewallCmdPath, "--zone="+zone, "--add-port="+arg); err != nil {
			return State{}, Error{Code: "PORT_OPEN_FAILED", Message: err.Error()}
		}
		if _, err := s.base.Runner.Run(ctx, s.cfg.FirewallCmdPath, "--permanent", "--zone="+zone, "--add-port="+arg); err != nil {
			_, _ = s.base.Runner.Run(ctx, s.cfg.FirewallCmdPath, "--zone="+zone, "--remove-port="+arg)
			return State{}, Error{Code: "PORT_OPEN_FAILED", Message: err.Error()}
		}
	}
	return s.LoadState(ctx)
}

func (s *CentOSService) ClosePort(ctx context.Context, request PortChangeRequest) (State, error) {
	req, err := ValidatePortChange(request)
	if err != nil {
		return State{}, err
	}
	specs, _ := ParsePortExpression(req.Port)
	zone, err := s.zone(ctx)
	if err != nil {
		return State{}, err
	}
	for _, spec := range specs {
		arg := firewalldPortArg(spec, req.Protocol)
		if _, err := s.base.Runner.Run(ctx, s.cfg.FirewallCmdPath, "--zone="+zone, "--remove-port="+arg); err != nil {
			return State{}, Error{Code: "PORT_CLOSE_FAILED", Message: err.Error()}
		}
		if _, err := s.base.Runner.Run(ctx, s.cfg.FirewallCmdPath, "--permanent", "--zone="+zone, "--remove-port="+arg); err != nil {
			_, _ = s.base.Runner.Run(ctx, s.cfg.FirewallCmdPath, "--zone="+zone, "--add-port="+arg)
			return State{}, Error{Code: "PORT_CLOSE_FAILED", Message: err.Error()}
		}
	}
	return s.LoadState(ctx)
}

func (s *CentOSService) zone(ctx context.Context) (string, error) {
	if s.cfg.CentOSZone != "" {
		return s.cfg.CentOSZone, nil
	}
	result, err := s.base.Runner.Run(ctx, s.cfg.FirewallCmdPath, "--get-default-zone")
	if err != nil {
		return "", Error{Code: "FIREWALL_STATE_LOAD_FAILED", Message: result.Stderr}
	}
	zone := strings.TrimSpace(result.Stdout)
	if zone == "" {
		return "", Error{Code: "FIREWALL_STATE_LOAD_FAILED", Message: "empty default zone"}
	}
	return zone, nil
}

func ParseFirewalldPorts(output string) []PortRule {
	ports := []PortRule{}
	for _, token := range strings.Fields(output) {
		if rule, ok := parsePortToken(token); ok {
			ports = append(ports, rule)
		}
	}
	return sortPorts(ports)
}
