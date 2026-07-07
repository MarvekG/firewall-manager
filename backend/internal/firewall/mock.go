package firewall

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"
)

type MockService struct {
	mu    sync.Mutex
	ports []PortRule
}

func NewMockService() *MockService {
	return &MockService{ports: []PortRule{{Port: 22, Protocol: "tcp", Source: "Any", Description: "SSH"}}}
}

func (s *MockService) LoadState(ctx context.Context) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state(), nil
}

func (s *MockService) OpenPort(ctx context.Context, request PortChangeRequest) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	protocol := strings.ToLower(request.Protocol)
	for _, port := range s.ports {
		if port.Port == request.Port && port.Protocol == protocol {
			return s.state(), nil
		}
	}
	s.ports = append(s.ports, PortRule{Port: request.Port, Protocol: protocol, Source: "Any"})
	return s.state(), nil
}

func (s *MockService) ClosePort(ctx context.Context, request PortChangeRequest) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	protocol := strings.ToLower(request.Protocol)
	s.ports = slices.DeleteFunc(s.ports, func(port PortRule) bool {
		return port.Port == request.Port && port.Protocol == protocol
	})
	return s.state(), nil
}

func (s *MockService) state() State {
	ports := slices.Clone(s.ports)
	slices.SortFunc(ports, func(a, b PortRule) int {
		if a.Port != b.Port {
			return a.Port - b.Port
		}
		return strings.Compare(a.Protocol, b.Protocol)
	})
	return State{
		OSType:                "development",
		Backend:               "mock",
		ServiceEnabled:        true,
		ServiceRunning:        true,
		DefaultIncomingPolicy: "deny",
		OpenPorts:             ports,
		LoadedAt:              time.Now().UTC(),
	}
}
