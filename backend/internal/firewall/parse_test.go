package firewall

import "testing"

func TestParseFirewalldPorts(t *testing.T) {
	ports := ParseFirewalldPorts("22/tcp 443/tcp 53/udp 1000-1005/tcp")
	if len(ports) != 3 {
		t.Fatalf("expected 3 ports, got %d", len(ports))
	}
	assertPort(t, ports[0], 22, "tcp")
	assertPort(t, ports[1], 53, "udp")
	assertPort(t, ports[2], 443, "tcp")
}

func TestParseUFWPorts(t *testing.T) {
	output := `Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), disabled (routed)

To                         Action      From
--                         ------      ----
22/tcp                     ALLOW       Anywhere
443/tcp                    ALLOW       10.0.0.0/8
53/udp                     ALLOW       Anywhere
22/tcp (v6)                ALLOW       Anywhere (v6)
`
	ports := ParseUFWPorts(output)
	if len(ports) != 3 {
		t.Fatalf("expected 3 ports, got %d: %#v", len(ports), ports)
	}
	assertPort(t, ports[0], 22, "tcp")
	assertPort(t, ports[1], 53, "udp")
	assertPort(t, ports[2], 443, "tcp")
	if ports[2].Source != "10.0.0.0/8" {
		t.Fatalf("expected source to be parsed, got %q", ports[2].Source)
	}
}

func TestParseUFWDefaultPolicy(t *testing.T) {
	policy := ParseUFWDefaultPolicy("Default: deny (incoming), allow (outgoing), disabled (routed)")
	if policy != "deny" {
		t.Fatalf("expected deny, got %q", policy)
	}
}

func TestValidatePortChange(t *testing.T) {
	if _, err := ValidatePortChange(PortChangeRequest{Port: 443, Protocol: "TCP"}); err != nil {
		t.Fatalf("expected valid request: %v", err)
	}
	if _, err := ValidatePortChange(PortChangeRequest{Port: 0, Protocol: "tcp"}); err == nil {
		t.Fatalf("expected invalid port")
	}
	if _, err := ValidatePortChange(PortChangeRequest{Port: 443, Protocol: "icmp"}); err == nil {
		t.Fatalf("expected invalid protocol")
	}
}

func assertPort(t *testing.T, rule PortRule, port int, protocol string) {
	t.Helper()
	if rule.Port != port || rule.Protocol != protocol {
		t.Fatalf("expected %d/%s, got %d/%s", port, protocol, rule.Port, rule.Protocol)
	}
}
