package firewall

import (
	"slices"
	"strconv"
	"strings"
)

func sortPorts(ports []PortRule) []PortRule {
	slices.SortFunc(ports, func(a, b PortRule) int {
		aStart := sortPortStart(a.Port)
		bStart := sortPortStart(b.Port)
		if aStart != bStart {
			return aStart - bStart
		}
		return strings.Compare(a.Protocol, b.Protocol)
	})
	return ports
}

func sortPortStart(value string) int {
	value = strings.ReplaceAll(value, ":", "-")
	parts := strings.Split(value, "-")
	port, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return port
}
