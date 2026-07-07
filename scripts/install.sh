#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_SOURCE="$INSTALL_DIR/firewall-manager"
ADMIN_USER=""
ADMIN_PASSWORD=""
LISTEN_HOST="127.0.0.1"
LISTEN_PORT="10240"
TLS_ENABLED="false"
ALLOW_INSECURE_REMOTE="false"
FIREWALL_BACKEND=""
USE_SUDO="true"
FIREWALL_ZONE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --admin-user) ADMIN_USER="$2"; shift 2 ;;
    --admin-password) ADMIN_PASSWORD="$2"; shift 2 ;;
    --listen-host) LISTEN_HOST="$2"; shift 2 ;;
    --listen-port) LISTEN_PORT="$2"; shift 2 ;;
    --tls-cert) TLS_CERT="$2"; TLS_ENABLED="true"; shift 2 ;;
    --tls-key) TLS_KEY="$2"; TLS_ENABLED="true"; shift 2 ;;
    --no-tls) TLS_ENABLED="false"; shift ;;
    --allow-insecure-remote) ALLOW_INSECURE_REMOTE="true"; shift ;;
    --firewall-zone) FIREWALL_ZONE="$2"; shift 2 ;;
    --no-sudo) USE_SUDO="false"; shift ;;
    *) printf 'Unknown option: %s\n' "$1" >&2; exit 2 ;;
  esac
done

if [[ ! -f "$BIN_SOURCE" ]]; then
  printf 'Missing binary: %s\n' "$BIN_SOURCE" >&2
  exit 1
fi

if [[ "$(id -u)" -ne 0 ]]; then
  printf 'Run as root.\n' >&2
  exit 1
fi

random_token() {
  openssl rand -base64 18 2>/dev/null | tr -d '=+/[:space:]' | cut -c1-20
}

if [[ -z "$ADMIN_USER" ]]; then
  ADMIN_USER="fm-$(random_token)"
fi
if [[ -z "$ADMIN_PASSWORD" ]]; then
  ADMIN_PASSWORD="$(random_token)$(random_token)"
fi

has_command() {
  PATH=/usr/sbin:/usr/bin:/sbin:/bin command -v "$1" >/dev/null 2>&1
}

detect_firewall_backend() {
  if has_command firewall-cmd; then
    printf 'firewalld'
    return 0
  fi
  if has_command ufw; then
    printf 'ufw'
    return 0
  fi

  return 1
}

if ! FIREWALL_BACKEND="$(detect_firewall_backend)"; then
  printf 'No supported firewall backend found. Install ufw or firewalld/firewall-cmd first.\n' >&2
  exit 1
fi

id -u firewall-manager >/dev/null 2>&1 || useradd --system --home-dir /var/lib/firewall-manager --shell /usr/sbin/nologin firewall-manager
mkdir -p /etc/firewall-manager /var/lib/firewall-manager /var/log/firewall-manager
install -m 0755 "$BIN_SOURCE" /usr/local/bin/firewall-manager

SESSION_SECRET="$(openssl rand -base64 32 2>/dev/null || date +%s%N)"
cat >/etc/firewall-manager/env <<EOF
FIREWALL_MANAGER_HOST=$LISTEN_HOST
FIREWALL_MANAGER_PORT=$LISTEN_PORT
FIREWALL_MANAGER_TLS_ENABLED=$TLS_ENABLED
FIREWALL_MANAGER_ALLOW_INSECURE_REMOTE=$ALLOW_INSECURE_REMOTE
FIREWALL_MANAGER_ADMIN_USER=$ADMIN_USER
FIREWALL_MANAGER_ADMIN_PASSWORD=$ADMIN_PASSWORD
FIREWALL_MANAGER_SESSION_SECRET=$SESSION_SECRET
FIREWALL_MANAGER_FIREWALL_BACKEND=$FIREWALL_BACKEND
FIREWALL_MANAGER_USE_SUDO=$USE_SUDO
FIREWALL_MANAGER_FIREWALL_ZONE=$FIREWALL_ZONE
EOF
chmod 0600 /etc/firewall-manager/env
chown root:firewall-manager /etc/firewall-manager/env

if [[ "${TLS_CERT:-}" != "" ]]; then
  printf 'FIREWALL_MANAGER_TLS_CERT=%s\n' "$TLS_CERT" >>/etc/firewall-manager/env
fi
if [[ "${TLS_KEY:-}" != "" ]]; then
  printf 'FIREWALL_MANAGER_TLS_KEY=%s\n' "$TLS_KEY" >>/etc/firewall-manager/env
fi

install -m 0644 "$INSTALL_DIR/deploy/firewall-manager.service" /etc/systemd/system/firewall-manager.service
case "$FIREWALL_BACKEND" in
  ufw)
    install -m 0440 "$INSTALL_DIR/deploy/sudoers.ufw" /etc/sudoers.d/firewall-manager
    ;;
  firewalld)
    install -m 0440 "$INSTALL_DIR/deploy/sudoers.firewalld" /etc/sudoers.d/firewall-manager
    ;;
  *)
    printf 'Unsupported firewall backend: %s\n' "$FIREWALL_BACKEND" >&2
    exit 2
    ;;
esac
if [[ -f /etc/sudoers.d/firewall-manager ]]; then
  visudo -cf /etc/sudoers.d/firewall-manager >/dev/null
fi
systemctl daemon-reload
systemctl enable --now firewall-manager

printf 'Firewall Manager installed.\n'
printf 'URL: %s://%s:%s\n' "$([[ "$TLS_ENABLED" == "true" ]] && printf https || printf http)" "$LISTEN_HOST" "$LISTEN_PORT"
printf 'User: %s\n' "$ADMIN_USER"
printf 'Password: %s\n' "$ADMIN_PASSWORD"
