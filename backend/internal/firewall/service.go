package firewall

import (
	"context"
	"time"
)

type State struct {
	OSType                string     `json:"osType"`
	Backend               string     `json:"backend"`
	ServiceEnabled        bool       `json:"serviceEnabled"`
	ServiceRunning        bool       `json:"serviceRunning"`
	DefaultIncomingPolicy string     `json:"defaultIncomingPolicy"`
	OpenPorts             []PortRule `json:"openPorts"`
	LoadedAt              time.Time  `json:"loadedAt"`
}

type PortRule struct {
	Port        string `json:"port"`
	Protocol    string `json:"protocol"`
	Source      string `json:"source,omitempty"`
	Description string `json:"description,omitempty"`
}

type PortChangeRequest struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
}

type Service interface {
	LoadState(ctx context.Context) (State, error)
	OpenPort(ctx context.Context, request PortChangeRequest) (State, error)
	ClosePort(ctx context.Context, request PortChangeRequest) (State, error)
}
