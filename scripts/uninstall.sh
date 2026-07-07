#!/usr/bin/env bash
set -euo pipefail

KEEP_DATA="false"
CONFIG_FILE="/etc/firewall-manager/env"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --keep-data) KEEP_DATA="true"; shift ;;
    *) printf 'Unknown option: %s\n' "$1" >&2; exit 2 ;;
  esac
done

if [[ "$(id -u)" -ne 0 ]]; then
  printf 'Run as root.\n' >&2
  exit 1
fi

has_command() {
  PATH=/usr/sbin:/usr/bin:/sbin:/bin command -v "$1" >/dev/null 2>&1
}

read_env_value() {
  local name="$1"
  local file="$2"
  local key value
  [[ -f "$file" ]] || return 0
  while IFS='=' read -r key value; do
    if [[ "$key" == "$name" ]]; then
      printf '%s' "$value"
      return 0
    fi
  done <"$file"
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

close_firewall_port() {
  local port="$1"
  local backend="$2"
  local zone="$3"
  [[ "$port" =~ ^[0-9]+$ ]] || return 0
  case "$backend" in
    ufw)
      has_command ufw || return 0
      ufw delete allow "${port}/tcp" >/dev/null 2>&1 || true
      ;;
    firewalld)
      has_command firewall-cmd || return 0
      local zone_arg=""
      if [[ -n "$zone" ]]; then
        zone_arg="--zone=$zone"
      fi
      firewall-cmd $zone_arg --remove-port="${port}/tcp" >/dev/null 2>&1 || true
      firewall-cmd --permanent $zone_arg --remove-port="${port}/tcp" >/dev/null 2>&1 || true
      ;;
  esac
}

INSTALLED_PORT="$(read_env_value FIREWALL_MANAGER_PORT "$CONFIG_FILE")"
INSTALLED_BACKEND="$(read_env_value FIREWALL_MANAGER_FIREWALL_BACKEND "$CONFIG_FILE")"
INSTALLED_ZONE="$(read_env_value FIREWALL_MANAGER_FIREWALL_ZONE "$CONFIG_FILE")"
if [[ -z "$INSTALLED_BACKEND" ]]; then
  INSTALLED_BACKEND="$(detect_firewall_backend || true)"
fi

close_firewall_port "$INSTALLED_PORT" "$INSTALLED_BACKEND" "$INSTALLED_ZONE"

systemctl disable --now firewall-manager >/dev/null 2>&1 || true

for pid in $(pgrep -f '^/usr/local/bin/firewall-manager($| )' 2>/dev/null || true); do
  kill "$pid" >/dev/null 2>&1 || true
done
sleep 1
for pid in $(pgrep -f '^/usr/local/bin/firewall-manager($| )' 2>/dev/null || true); do
  kill -9 "$pid" >/dev/null 2>&1 || true
done

rm -f /etc/systemd/system/firewall-manager.service
rm -f /etc/sudoers.d/firewall-manager
rm -f /usr/local/bin/firewall-manager
systemctl daemon-reload

if [[ "$KEEP_DATA" != "true" ]]; then
  rm -rf /etc/firewall-manager /var/lib/firewall-manager /var/log/firewall-manager
  userdel firewall-manager >/dev/null 2>&1 || true
fi

printf 'Firewall Manager uninstalled.\n'
