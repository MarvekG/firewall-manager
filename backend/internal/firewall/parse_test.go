package firewall

import "testing"

func TestParseFirewalldPorts(t *testing.T) {
	ports := ParseFirewalldPorts("22/tcp 443/tcp 53/udp 1000-1005/tcp")
	if len(ports) != 4 {
		t.Fatalf("expected 3 ports, got %d", len(ports))
	}
	assertPort(t, ports[0], "22", "tcp")
	assertPort(t, ports[1], "53", "udp")
	assertPort(t, ports[2], "443", "tcp")
	assertPort(t, ports[3], "1000-1005", "tcp")
}

func TestParseUFWPorts(t *testing.T) {
	output := `Status: active
Logging: on (low)
Default: deny (incoming), allow (outgoing), disabled (routed)

To                         Action      From
--                         ------      ----
22/tcp                     ALLOW       Anywhere
443/tcp                    ALLOW       10.0.0.0/8
1000:1005/tcp              ALLOW       Anywhere
53/udp                     ALLOW       Anywhere
22/tcp (v6)                ALLOW       Anywhere (v6)
`
	ports := ParseUFWPorts(output)
	if len(ports) != 4 {
		t.Fatalf("expected 3 ports, got %d: %#v", len(ports), ports)
	}
	assertPort(t, ports[0], "22", "tcp")
	assertPort(t, ports[1], "53", "udp")
	assertPort(t, ports[2], "443", "tcp")
	if ports[2].Source != "10.0.0.0/8" {
		t.Fatalf("expected source to be parsed, got %q", ports[2].Source)
	}
	assertPort(t, ports[3], "1000-1005", "tcp")
}

func TestParseUFWDefaultPolicy(t *testing.T) {
	policy := ParseUFWDefaultPolicy("Default: deny (incoming), allow (outgoing), disabled (routed)")
	if policy != "deny" {
		t.Fatalf("expected deny, got %q", policy)
	}
}

func TestValidatePortChange(t *testing.T) {
	request, err := ValidatePortChange(PortChangeRequest{Port: "443, 1000:1005", Protocol: "TCP"})
	if err != nil {
		t.Fatalf("expected valid request: %v", err)
	}
	if request.Port != "443,1000-1005" || request.Protocol != "tcp" {
		t.Fatalf("expected normalized request, got %#v", request)
	}
	if _, err := ValidatePortChange(PortChangeRequest{Port: "0", Protocol: "tcp"}); err == nil {
		t.Fatalf("expected invalid port")
	}
	if _, err := ValidatePortChange(PortChangeRequest{Port: "443", Protocol: "icmp"}); err == nil {
		t.Fatalf("expected invalid protocol")
	}
	if _, err := ValidatePortChange(PortChangeRequest{Port: "100-99", Protocol: "tcp"}); err == nil {
		t.Fatalf("expected invalid range")
	}
}

func assertPort(t *testing.T, rule PortRule, port string, protocol string) {
	t.Helper()
	if rule.Port != port || rule.Protocol != protocol {
		t.Fatalf("expected %s/%s, got %s/%s", port, protocol, rule.Port, rule.Protocol)
	}
}
