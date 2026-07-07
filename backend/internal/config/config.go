package config

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Auth     AuthConfig
	Firewall FirewallConfig
}

type ServerConfig struct {
	Host                string
	Port                int
	PublicURL           string
	AllowInsecureRemote bool
	TLS                 TLSConfig
}

type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

type AuthConfig struct {
	AdminUser     string
	AdminPassword string
	SessionSecret string
	SessionTTL    time.Duration
}

type FirewallConfig struct {
	Backend         string
	UseSudo         bool
	CommandTimeout  time.Duration
	CentOSZone      string
	FirewallCmdPath string
	UFWPath         string
	SystemctlPath   string
}

func Load() Config {
	port := envInt("FIREWALL_MANAGER_PORT", 10240)
	tlsEnabled := envBool("FIREWALL_MANAGER_TLS_ENABLED", false)
	host := env("FIREWALL_MANAGER_HOST", "127.0.0.1")
	publicURL := env("FIREWALL_MANAGER_PUBLIC_URL", defaultPublicURL(host, port, tlsEnabled))

	return Config{
		Server: ServerConfig{
			Host:                host,
			Port:                port,
			PublicURL:           publicURL,
			AllowInsecureRemote: envBool("FIREWALL_MANAGER_ALLOW_INSECURE_REMOTE", false),
			TLS: TLSConfig{
				Enabled:  tlsEnabled,
				CertFile: env("FIREWALL_MANAGER_TLS_CERT", "/etc/firewall-manager/tls.crt"),
				KeyFile:  env("FIREWALL_MANAGER_TLS_KEY", "/etc/firewall-manager/tls.key"),
			},
		},
		Auth: AuthConfig{
			AdminUser:     env("FIREWALL_MANAGER_ADMIN_USER", "admin"),
			AdminPassword: env("FIREWALL_MANAGER_ADMIN_PASSWORD", "admin"),
			SessionSecret: env("FIREWALL_MANAGER_SESSION_SECRET", randomSecret()),
			SessionTTL:    time.Duration(envInt("FIREWALL_MANAGER_SESSION_TTL_MINUTES", 480)) * time.Minute,
		},
		Firewall: FirewallConfig{
			Backend:         env("FIREWALL_MANAGER_FIREWALL_BACKEND", "auto"),
			UseSudo:         envBool("FIREWALL_MANAGER_USE_SUDO", true),
			CommandTimeout:  time.Duration(envInt("FIREWALL_MANAGER_COMMAND_TIMEOUT_SECONDS", 10)) * time.Second,
			CentOSZone:      env("FIREWALL_MANAGER_FIREWALL_ZONE", ""),
			FirewallCmdPath: env("FIREWALL_MANAGER_FIREWALL_CMD", "firewall-cmd"),
			UFWPath:         env("FIREWALL_MANAGER_UFW", "ufw"),
			SystemctlPath:   env("FIREWALL_MANAGER_SYSTEMCTL", "systemctl"),
		},
	}
}

func (s ServerConfig) Addr() string {
	return net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
}

func (s ServerConfig) IsLoopback() bool {
	if s.Host == "localhost" {
		return true
	}
	ip := net.ParseIP(s.Host)
	return ip != nil && ip.IsLoopback()
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func defaultPublicURL(host string, port int, tls bool) string {
	scheme := "http"
	if tls {
		scheme = "https"
	}
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(port))
}

func randomSecret() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "development-session-secret-change-me"
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}
