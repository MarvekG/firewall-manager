# Firewall Manager

中文 | [English](README.en.md)

一个轻量、现代、易部署的服务器防火墙端口管理面板。

Firewall Manager 使用 Go 后端和 Vite React 前端。源码层前后端分离，生产环境只运行一个低权限 Go 进程，同时提供静态前端页面和 `/api/*` 后端接口。

## 核心优势

- 单进程部署：生产环境只运行一个 `firewall-manager` 二进制。
- 资源占用低：不需要 Nginx、Caddy、Node.js、Vite dev server、Python runtime 或数据库。
- 默认低权限运行：systemd 服务使用专用 `firewall-manager` 用户。
- 受控提权：只通过按后端区分的 sudoers 文件执行必要的 `ufw` 或 `firewall-cmd` 命令。
- 现代 UI：Vite + React + TypeScript + Tailwind CSS。
- 前后端分离体验：开发期前端和后端独立运行，生产期前端静态资源由 Go 二进制托管。
- 支持 Ubuntu/UFW 和 CentOS/firewalld。
- 支持 HTTPS/TLS，也支持本机或可信内网 HTTP 模式。
- 支持中文和英文界面。
- 不支持注册：管理员账号通过部署配置创建，减少公开攻击面。
- firewalld 安全策略明确：不使用 `firewall-cmd --reload`、`--complete-reload` 或 `--runtime-to-permanent`。

## 当前功能

- 登录/退出登录。
- 签名 session cookie。
- 运行时 TLS/HTTP 安全提示。
- 加载防火墙状态。
- 展示系统摘要。
- 展示开放端口列表。
- 打开端口。
- 关闭端口，带确认弹窗。
- Ubuntu/UFW 适配。
- CentOS/firewalld 适配。
- 一键安装、重新安装、卸载脚本。

## 架构

```text
Browser
  |
  | HTTP/HTTPS
  v
firewall-manager，Go 单进程，低权限用户运行
  |
  +-- Static Web
  |     +-- Vite build 后的 React 静态资源
  |
  +-- HTTP API
        +-- Auth / Session
        +-- Runtime config
        +-- Firewall API
  |
  v
FirewallService
  +-- UFWService，ufw
  +-- FirewalldService，firewall-cmd

高权限命令通过 sudoers 控制。
```

## 技术栈

- 后端：Go `net/http`
- 前端：Vite + React + TypeScript + Tailwind CSS
- 日志：Go `log/slog`
- 部署：单二进制 + systemd
- 提权：按后端区分的 sudoers 文件

## 默认端口

```text
10240
```

开发访问：

```text
http://127.0.0.1:10240
```

## 本地开发

后端：

```bash
cd backend
go run ./cmd/firewall-manager
```

后端启动需要当前节点存在 `firewall-cmd` 或 `ufw` 命令，并会按命令自动选择真实防火墙后端。

前端：

```bash
cd frontend
npm install
npm run dev
```

前端开发地址：

```text
http://127.0.0.1:5173
```

本地开发未设置认证环境变量时，后端回退到以下账号：

```text
admin / admin
```

正式安装时，管理员账号以安装脚本输出或 `--admin-user` / `--admin-password` 参数为准。

## 构建

```bash
bash scripts/build-release.sh
```

发布产物在：

```text
dist/
```

发布包包含：

```text
dist/firewall-manager
dist/install.sh
dist/reinstall.sh
dist/uninstall.sh
dist/deploy/firewall-manager.service
dist/deploy/sudoers.ufw
dist/deploy/sudoers.firewalld
```

## 安装 Ubuntu/UFW

安装脚本只按节点上的命令自动选择后端：检测到 `firewall-cmd` 时使用 firewalld，否则检测到 `ufw` 时使用 UFW。
安装时会开放 `--listen-port` 指定的 TCP 端口，默认是 `10240/tcp`。

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

安装脚本会安装 `deploy/sudoers.ufw` 到：

```text
/etc/sudoers.d/firewall-manager
```

## 安装 CentOS/firewalld

安装脚本只按节点上的命令自动选择后端：检测到 `firewall-cmd` 时使用 firewalld，否则检测到 `ufw` 时使用 UFW。
安装时会开放 `--listen-port` 指定的 TCP 端口，默认是 `10240/tcp`。

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

安装脚本会安装 `deploy/sudoers.firewalld` 到：

```text
/etc/sudoers.d/firewall-manager
```

## HTTPS/TLS

使用已有证书：

```bash
sudo ./install.sh \
  --listen-host 0.0.0.0 \
  --listen-port 10240 \
  --admin-user admin \
  --admin-password 'change-this-password' \
  --tls-cert /etc/firewall-manager/tls.crt \
  --tls-key /etc/firewall-manager/tls.key
```

HTTP 模式下 session cookie 不设置 `Secure`。HTTPS 模式下 session cookie 设置 `Secure=true`。

## 重新安装

保留配置和数据，重新安装二进制、systemd 和 sudoers：

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

## 卸载

卸载时会读取 `/etc/firewall-manager/env`，关闭安装时使用的监听 TCP 端口。

完全卸载：

```bash
cd dist
sudo ./uninstall.sh
```

保留配置、数据、日志和系统用户：

```bash
cd dist
sudo ./uninstall.sh --keep-data
```

## 验证

```bash
cd backend
go test ./...

cd ../frontend
npm run build

cd ..
bash scripts/build-release.sh
```

## 安全说明

- 请优先使用 HTTPS。
- 如果使用 HTTP，请仅在本机或可信内网中使用。
- 不要把默认开发密码用于生产。
- sudoers 文件是成品部署文件，但 sudoers 通配是 glob 风格；应用层仍会对端口和协议做严格校验。
- firewalld 适配器不会执行 `firewall-cmd --reload`，避免丢失 runtime-only 规则。
- firewalld 适配器不会执行 `firewall-cmd --runtime-to-permanent`，避免把临时规则整体固化。

## 许可证

MIT
