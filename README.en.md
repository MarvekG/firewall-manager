# Firewall Manager

[中文](README.md) | English

A lightweight, modern, easy-to-deploy web console for managing server firewall ports.

Firewall Manager uses a Go backend and a Vite React frontend. The source code is frontend/backend separated, while production runs as a single low-privilege Go process serving both static frontend assets and `/api/*` backend endpoints.

## Key Advantages

- Single-process deployment: only one `firewall-manager` binary runs in production.
- Low resource usage: no Nginx, Caddy, Node.js, Vite dev server, Python runtime, or database required in production.
- Low-privilege by default: the systemd service runs as a dedicated `firewall-manager` user.
- Controlled privilege elevation: backend-specific sudoers files allow only the required `ufw` or `firewall-cmd` command shapes.
- Modern UI: Vite + React + TypeScript + Tailwind CSS.
- Frontend/backend separation for development: frontend and backend run independently during development, while production serves frontend assets from the Go binary.
- Supports Ubuntu/UFW and CentOS/firewalld.
- Supports HTTPS/TLS and local/trusted-network HTTP mode.
- Supports Chinese and English UI.
- No public registration: admin accounts are configured during deployment.
- Explicit firewalld safety policy: no `firewall-cmd --reload`, no `--complete-reload`, and no `--runtime-to-permanent`.

## Features

- Sign in and sign out.
- Signed session cookie.
- Runtime TLS/HTTP security warning.
- Load firewall state.
- Show system summary.
- Show open ports.
- Open ports.
- Close ports with confirmation.
- Ubuntu/UFW adapter.
- CentOS/firewalld adapter.
- Install, reinstall, and uninstall scripts.

## Architecture

```text
Browser
  |
  | HTTP/HTTPS
  v
firewall-manager, single low-privilege Go process
  |
  +-- Static Web
  |     +-- React assets built by Vite
  |
  +-- HTTP API
        +-- Auth / Session
        +-- Runtime config
        +-- Firewall API
  |
  v
FirewallService
  +-- UbuntuFirewallService, ufw
  +-- CentOSFirewallService, firewall-cmd

Privileged commands are controlled through sudoers.
```

## Tech Stack

- Backend: Go `net/http`
- Frontend: Vite + React + TypeScript + Tailwind CSS
- Logs: Go `log/slog`
- Deployment: single binary + systemd
- Privilege elevation: backend-specific sudoers files

## Default Port

```text
10240
```

## Local Development

Backend:

```bash
cd backend
go run ./cmd/firewall-manager
```

The backend requires `firewall-cmd` or `ufw` on the node and automatically selects the real firewall backend by command availability.

Frontend:

```bash
cd frontend
npm install
npm run dev
```

When local development auth environment variables are not set, the backend falls back to:

```text
admin / admin
```

For production installs, use the credentials printed by the installer or set `--admin-user` / `--admin-password` explicitly.

## Build

```bash
bash scripts/build-release.sh
```

Release files are generated in `dist/`.

## Install on Ubuntu/UFW

The installer selects the backend only from commands available on the node: `firewall-cmd` means firewalld, otherwise `ufw` means UFW.

```bash
cd dist
sudo ./install.sh \
  --listen-host 0.0.0.0 \
  --listen-port 10240 \
  --admin-user admin \
  --admin-password 'change-this-password' \
  --no-tls \
  --allow-insecure-remote
```

## Install on CentOS/firewalld

The installer selects the backend only from commands available on the node: `firewall-cmd` means firewalld, otherwise `ufw` means UFW.

```bash
cd dist
sudo ./install.sh \
  --listen-host 0.0.0.0 \
  --listen-port 10240 \
  --firewall-zone public \
  --admin-user admin \
  --admin-password 'change-this-password' \
  --no-tls \
  --allow-insecure-remote
```

## Reinstall

```bash
cd dist
sudo ./reinstall.sh \
  --listen-host 0.0.0.0 \
  --listen-port 10240 \
  --admin-user admin \
  --admin-password 'change-this-password' \
  --no-tls \
  --allow-insecure-remote
```

## Uninstall

```bash
cd dist
sudo ./uninstall.sh
```

Keep data:

```bash
cd dist
sudo ./uninstall.sh --keep-data
```

## Security Notes

- Prefer HTTPS.
- Use HTTP only on localhost or trusted networks.
- Never use the development fallback credentials in production.
- Sudoers files are production files, but sudoers wildcards are glob-style. The application still validates port and protocol strictly before command execution.
- The firewalld adapter never runs `firewall-cmd --reload`.
- The firewalld adapter never runs `firewall-cmd --runtime-to-permanent`.

## License

MIT
