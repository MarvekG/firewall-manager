package firewall

import (
	"context"
	"slices"
	"sync"
	"time"
)

type MockService struct {
	mu    sync.Mutex
	ports []PortRule
}

func NewMockService() *MockService {
	return &MockService{ports: []PortRule{{Port: "22", Protocol: "tcp", Source: "Any", Description: "SSH"}}}
}

func (s *MockService) LoadState(ctx context.Context) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state(), nil
}

func (s *MockService) OpenPort(ctx context.Context, request PortChangeRequest) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, err := ValidatePortChange(request)
	if err != nil {
		return State{}, err
	}
	specs, _ := ParsePortExpression(req.Port)
	for _, spec := range specs {
		exists := false
		for _, port := range s.ports {
			if port.Port == spec.Value && port.Protocol == req.Protocol {
				exists = true
				break
			}
		}
		if !exists {
			s.ports = append(s.ports, PortRule{Port: spec.Value, Protocol: req.Protocol, Source: "Any"})
		}
	}
	return s.state(), nil
}

func (s *MockService) ClosePort(ctx context.Context, request PortChangeRequest) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, err := ValidatePortChange(request)
	if err != nil {
		return State{}, err
	}
	specs, _ := ParsePortExpression(req.Port)
	remove := map[string]bool{}
	for _, spec := range specs {
		remove[spec.Value] = true
	}
	s.ports = slices.DeleteFunc(s.ports, func(port PortRule) bool {
		return remove[port.Port] && port.Protocol == req.Protocol
	})
	return s.state(), nil
}

func (s *MockService) state() State {
	ports := slices.Clone(s.ports)
	return State{
		OSType:                "development",
		Backend:               "mock",
		ServiceEnabled:        true,
		ServiceRunning:        true,
		DefaultIncomingPolicy: "deny",
		OpenPorts:             sortPorts(ports),
		LoadedAt:              time.Now().UTC(),
	}
}
