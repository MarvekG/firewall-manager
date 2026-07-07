package firewall

import (
	"slices"
	"strings"
)

func sortPorts(ports []PortRule) []PortRule {
	slices.SortFunc(ports, func(a, b PortRule) int {
		if a.Port != b.Port {
			return a.Port - b.Port
		}
		return strings.Compare(a.Protocol, b.Protocol)
	})
	return ports
}
