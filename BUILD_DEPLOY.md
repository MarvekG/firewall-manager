# 构建与部署指南

## 架构

Firewall Manager 使用源码层前后端分离、部署层单进程架构。

```text
frontend/  Vite + React + TypeScript + Tailwind
backend/   Go net/http API + 静态资源托管
dist/      发布产物
```

生产环境只运行一个 `firewall-manager` 二进制：

```text
/       React 静态页面
/api/*  Go 后端 API
```

生产环境不需要 Node.js、Vite dev server、Nginx 或 Caddy。

## 本地开发

后端：

```bash
cd backend
go run ./cmd/firewall-manager
```

后端启动需要当前节点存在 `firewall-cmd` 或 `ufw` 命令，并会按命令自动选择真实防火墙后端。

访问：

```text
http://127.0.0.1:10240
```

前端：

```bash
cd frontend
npm install
npm run dev
```

访问：

```text
http://127.0.0.1:5173
```

本地开发未设置认证环境变量时，后端回退到以下账号：

```text
admin / admin
```

正式安装时，管理员账号以安装脚本输出或 `--admin-user` / `--admin-password` 参数为准。

## 构建发布包

构建机需要 Go、nvm、最新版 Node.js 和 npm。

Ubuntu/Debian 构建机：

```bash
sudo apt update
sudo apt install -y golang-go curl ca-certificates libatomic1
```

CentOS/RHEL/Rocky/Alma/Fedora 构建机：

```bash
sudo dnf install -y golang curl ca-certificates libatomic
```

安装 nvm 和最新版 Node.js：

```bash
curl -fsSL https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.5/install.sh | bash
. "$HOME/.nvm/nvm.sh"
nvm install node
nvm use node
node --version
npm --version
```

如果 `node --version` 报错缺少 `libatomic.so.1`，说明系统没有安装 Node 所需的 `libatomic` 运行库：

```bash
# Ubuntu/Debian
sudo apt install -y libatomic1

# CentOS/RHEL/Rocky/Alma/Fedora
sudo dnf install -y libatomic

# CentOS 7
sudo yum install -y libatomic
```

```bash
bash scripts/build-release.sh
```

发布产物：

```text
dist/firewall-manager
dist/install.sh
dist/reinstall.sh
dist/uninstall.sh
dist/deploy/firewall-manager.service
dist/deploy/sudoers.ufw
dist/deploy/sudoers.firewalld
dist/deploy/config.example.yml
```

## 安装系统依赖

目标服务器不需要 Node.js、npm、Go 或 OpenSSL，只需要 systemd、sudo 和一个真实防火墙后端命令。

安装脚本只按节点上的命令自动选择后端：检测到 `firewall-cmd` 时使用 firewalld，否则检测到 `ufw` 时使用 UFW。两个命令都不存在时安装失败。

Ubuntu/Debian 使用 UFW：

```bash
sudo apt update
sudo apt install -y ufw sudo
sudo systemctl enable --now ufw
```

Ubuntu/Debian 使用 firewalld：

```bash
sudo apt update
sudo apt install -y firewalld sudo
sudo systemctl enable --now firewalld
sudo firewall-cmd --state
```

CentOS/RHEL/Rocky/Alma/Fedora 使用 firewalld：

```bash
sudo dnf install -y firewalld sudo
sudo systemctl enable --now firewalld
sudo firewall-cmd --state
```

CentOS 7 使用 firewalld：

```bash
sudo yum install -y firewalld sudo
sudo systemctl enable --now firewalld
sudo firewall-cmd --state
```

如果同一台机器同时安装了 `firewall-cmd` 和 `ufw`，会选择 firewalld。不要同时启用两个防火墙后端管理同一套规则。

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

安装的 sudoers 文件：

```text
/etc/sudoers.d/firewall-manager  <- deploy/sudoers.ufw
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

安装的 sudoers 文件：

```text
/etc/sudoers.d/firewall-manager  <- deploy/sudoers.firewalld
```

firewalld 版本不会使用：

```text
firewall-cmd --reload
firewall-cmd --complete-reload
firewall-cmd --runtime-to-permanent
```

## HTTPS/TLS 安装

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

保留数据和配置，重装二进制、systemd 和 sudoers：

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

完全卸载，包括配置、数据、日志和系统用户：

```bash
cd dist
sudo ./uninstall.sh
```

只卸载服务和二进制，保留配置、数据、日志和系统用户：

```bash
cd dist
sudo ./uninstall.sh --keep-data
```

## 维护命令

```bash
systemctl status firewall-manager
journalctl -u firewall-manager -f
systemctl restart firewall-manager
systemctl stop firewall-manager
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

启动发布二进制时会自动检测真实防火墙后端，当前节点需要存在 `firewall-cmd` 或 `ufw`：

```bash
FIREWALL_MANAGER_PORT=10240 \
./dist/firewall-manager
```
