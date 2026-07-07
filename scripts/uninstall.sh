#!/usr/bin/env bash
set -euo pipefail

KEEP_DATA="false"

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

systemctl disable --now firewall-manager >/dev/null 2>&1 || true
rm -f /etc/systemd/system/firewall-manager.service
rm -f /etc/sudoers.d/firewall-manager
rm -f /usr/local/bin/firewall-manager
systemctl daemon-reload

if [[ "$KEEP_DATA" != "true" ]]; then
  rm -rf /etc/firewall-manager /var/lib/firewall-manager /var/log/firewall-manager
  userdel firewall-manager >/dev/null 2>&1 || true
fi

printf 'Firewall Manager uninstalled.\n'
