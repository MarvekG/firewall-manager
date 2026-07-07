# 防火墙管理系统设计文档

## 1. 概述

防火墙管理系统是一个用于管理服务器防火墙端口的网站。系统默认以低权限用户运行，访问管理系统前必须先完成认证；认证通过后进入防火墙管理控制台。系统支持中文和英文界面，并通过统一的防火墙服务抽象适配 CentOS 和 Ubuntu。

第一版聚焦于管理当前服务器本机防火墙。远程管理多台服务器不属于第一版范围。

## 2. 目标

- 提供基于浏览器的防火墙管理控制台。
- 主 Web 应用默认以非 root 的低权限系统用户运行。
- 所有管理页面和管理 API 都必须先登录后访问。
- 不支持公开注册。
- 支持中文和英文国际化。
- 登录后加载并展示当前系统防火墙初始状态。
- 支持打开和关闭防火墙端口。
- 支持 CentOS 和 Ubuntu，通过一个共享 `FirewallService` 接口和通用组合结构实现两个系统适配器。
- 将高权限操作限制在很窄的边界内，便于审计和控制。

## 3. 非目标

- 用户自助注册。
- 多租户账号管理。
- 通过 SSH 管理远程主机。
- 高级防火墙规则编辑，例如 rich rules、源 IP 白名单、NAT、zone、chain 等。
- 实时协作编辑。
- 完整替代 `firewalld`、`ufw`、`iptables` 或 `nftables`。

## 4. 技术栈调研与推荐

### 4.1 评估维度

这个项目需要现代化 UI、前后端分离源码结构、一键部署，并且运行资源占用尽量低。技术栈按以下优先级评估：

1. 运行资源低：生产环境不运行 Node.js、Vite dev server、Nginx、Caddy、Python 解释器或多进程栈。
2. 部署简单：一键脚本安装一个后端可执行文件、一个 systemd 服务和必要配置。
3. UI 现代化：支持组件化、响应式布局、良好的表单和状态反馈。
4. 源码层前后端分离：前端独立开发和构建，后端只提供 API 和系统能力。
5. 部署层单进程：生产环境由同一个进程同时提供静态前端资源和 `/api/*`。
6. 权限边界清晰：Web 后端低权限运行，高权限防火墙操作通过受控 helper 或 sudoers。
7. 系统兼容：适配 Ubuntu 和 CentOS。
8. 可测试性：防火墙命令构造、命令输出解析、认证、i18n 和 API 都应可独立测试。

### 4.2 前后端分离的落地方式

Rust 或 Go 做现代网站时，仍然可以采用前后端分离，但这里的“分离”应指源码和职责分离，不应强制变成生产环境多服务部署。

推荐落地方式：

```text
开发期：
  frontend/ 使用 Vite 独立开发 UI
  backend/ 使用 Go 或 Rust 独立开发 API
  Vite dev server 通过 proxy 转发 /api 到后端

生产期：
  Vite build 生成 frontend/dist
  后端可执行文件内嵌或托管 frontend/dist
  单个 firewall-manager 进程同时处理：
    /       -> 前端静态文件
    /api/* -> 后端 API
```

这样既保留现代前端开发体验，也避免线上多进程和反向代理开销。

### 4.3 候选方案对比

| 方案 | 优点 | 缺点 | 结论 |
| --- | --- | --- | --- |
| Go API + Vite React TypeScript + Tailwind | 单二进制；资源占用低；部署简单；`net/http` 可同时托管静态文件和 API；开发成本适中 | 没有 class 继承语法；需要用接口和组合表达共享通用逻辑 | 首选 |
| Rust Axum API + Vite React TypeScript + Tailwind | 资源占用更极致；内存安全强；单二进制；适合长期系统工具 | 开发成本高；团队门槛更高；认证/session/CSRF 组合更费工 | 备选，如果优先绝对资源和安全 |
| Python FastAPI + Vite React | 后端开发最快；抽象类表达自然；解析文本方便 | 生产需要 Python 运行时和依赖环境；常驻资源占用更高 | 不作为低资源首选 |
| Node.js 全栈 | 前后端同语言；TypeScript 体验统一 | 生产需要 Node.js 运行时；系统管理工具场景资源占用偏高 | 不推荐 |

### 4.4 首选技术栈

第一版推荐使用 Go + Vite 的源码层前后端分离、部署层单进程架构：

- 后端语言：Go。
- HTTP 层：Go 标准库 `net/http`，优先使用 Go `ServeMux`。如果后续路由复杂，再引入 `github.com/go-chi/chi/v5`。
- 静态资源：使用 `embed` 将 `frontend/dist` 打进后端二进制，或一键部署时放到 `/usr/share/firewall-manager/web` 由同一进程托管。
- 配置文件：YAML，例如 `/etc/firewall-manager/config.yml`。
- 认证：用户名密码登录，Argon2id 密码哈希，HTTP-only session cookie。
- Session：首版使用签名 cookie，避免引入数据库或 Redis。
- CSRF：双提交 cookie 或服务端生成 CSRF token，前端通过 header 提交。
- 命令执行：`exec.CommandContext`，必须 argv 执行，不通过 shell，必须设置超时。
- 日志：Go `log/slog` 输出结构化 JSON 日志，审计日志可写 JSONL。
- 前端构建：Vite。
- 前端框架：React + TypeScript。
- 前端样式：Tailwind CSS。
- 前端组件：Radix UI primitives，可采用 shadcn/ui 风格组织组件；这些只影响构建产物，不增加生产进程。
- 前端 i18n：i18next 或轻量 JSON 字典，支持 `zh-CN` 和 `en-US`。
- 前端状态：首版可用 React state；如果接口状态变复杂，再引入 TanStack Query。
- 部署方式：一键安装脚本安装一个 `firewall-manager` 二进制、配置文件、systemd service 和 sudoers/helper。

生产环境不运行 Vite dev server。Vite 只用于开发和构建，构建产物由 `firewall-manager` 后端进程直接提供。

### 4.5 Rust 备选技术栈

如果目标是资源占用尽可能极致，并且可以接受更高开发成本，可以选择 Rust：

- 后端语言：Rust。
- Web 框架：Axum。
- Runtime：Tokio。
- 静态资源：`tower-http` `ServeDir` 或 `rust-embed`。
- 配置：`serde` + `config` 或 `figment`。
- 认证：`argon2` + 签名 cookie/session。
- 日志：`tracing`。
- 命令执行：`tokio::process::Command`，必须 argv 执行，不通过 shell。
- 前端：同样使用 Vite React TypeScript Tailwind。

Rust 版本仍然采用源码层前后端分离、部署层单进程。区别只是后端实现从 Go 换成 Rust。

### 4.6 推荐模块拆分

前端模块：

- `pages`：登录页、控制台页、错误页。
- `components`：端口表格、状态卡片、语言切换器、确认弹窗、表单控件。
- `api`：封装 `/api/*` 请求、统一错误码处理和 CSRF header。
- `i18n`：中英文资源和语言选择逻辑。
- `styles`：Tailwind 入口和全局样式。

后端模块：

- Auth 模块：登录、退出、当前用户、session、CSRF。
- Firewall 模块：提供与操作系统无关的 `FirewallService` 接口。
- Command Runner 模块：以 argv 形式执行受控命令，负责超时、审计和错误归一化。
- Config 模块：管理员账号、会话密钥、监听地址、默认语言、命令/helper 路径。
- Static Web 模块：托管 Vite 构建后的前端资源，并对前端路由回退到 `index.html`。
- API 模块：认证、语言、防火墙状态、端口开关。

防火墙模块要做成深模块：上层只调用 `LoadState`、`OpenPort` 和 `ClosePort`，不需要知道 firewalld 或 ufw 的命令细节。

### 4.7 开发和生产形态

开发环境：

```text
Vite dev server:       http://127.0.0.1:5173
Firewall Manager API:  http://127.0.0.1:10240
```

开发期前端通过 Vite proxy 调用后端 `/api/*`，避免跨域复杂度。

生产环境：

```text
firewall-manager 单进程，低权限用户运行
  -> /       前端静态文件，来自 Vite build 产物
  -> /api/* 后端 API
```

生产环境不引入 Nginx/Caddy，不运行 Node.js，不运行 Vite dev server。默认监听高位端口 `10240`，例如 `0.0.0.0:10240` 或 `127.0.0.1:10240`，避免低权限进程绑定 80/443。

### 4.8 项目目录结构

建议目录结构：

```text
firewall-manager/
  backend/
    cmd/
      firewall-manager/
        main.go
    internal/
      app/
      auth/
      command/
      config/
      firewall/
        service.go
        base.go
        centos.go
        ubuntu.go
        factory.go
      httpapi/
      i18n/
      staticweb/
    webdist/
      .gitkeep
    go.mod

  frontend/
    src/
      api/
      components/
      i18n/
      pages/
      styles/
      main.tsx
    package.json
    vite.config.ts
    tailwind.config.ts
    tsconfig.json

  scripts/
    install.sh
    build-release.sh

  deploy/
    firewall-manager.service
    sudoers.ufw
    sudoers.firewalld
    config.example.yml

  DESIGN.md
```

`scripts/build-release.sh` 负责先构建前端，再把 `frontend/dist` 复制到 `backend/webdist`，最后执行 `go build`。后端通过 Go `embed` 将 `backend/webdist` 打进二进制。

## 5. 总体架构

```text
Browser
  |
  | HTTP/HTTPS
  v
firewall-manager（Go 单进程，低权限用户）
  |
  +-- Static Web
  |     |
  |     +-- 托管 Vite build 后的 React 静态资源
  |
  +-- HTTP API
        |
        +-- Auth / Session / CSRF
        +-- Locale
        +-- Firewall API
  |
  | 已认证的 API 调用
  v
FirewallService 接口
  |
  +-- CentOSFirewallService
  |     |
  |     +-- firewalld/firewall-cmd 适配器
  |
  +-- UbuntuFirewallService
        |
        +-- ufw 适配器

高权限命令执行通过 sudoers 或受控 helper 限制在严格白名单内。
生产环境不需要 Nginx、Caddy、Node.js 或 Vite dev server。
```

## 6. 运行权限模型

### 6.1 默认运行用户

Web 进程以专用低权限用户运行：

```text
user: firewall-manager
group: firewall-manager
home: /var/lib/firewall-manager
config: /etc/firewall-manager/config.yml
```

正常处理 Web 请求时，进程不得以 `root` 身份运行。

### 6.2 高权限操作

打开和关闭端口需要高权限。可选方案：

1. 第一版推荐：通过 sudoers 配置命令白名单。
2. 后续可选：实现一个 root 拥有的小型 helper 二进制或守护进程，通过严格 JSON 协议通信。

使用 sudoers 时，只允许应用执行必要的防火墙命令。应用不得接受任意 shell 片段，也不得通过 shell 拼接执行命令。

概念示例：

```text
Ubuntu/UFW 使用 `deploy/sudoers.ufw`，CentOS/firewalld 使用 `deploy/sudoers.firewalld`。
```

最终 sudoers 配置应尽可能比上述示例更严格，限制到受支持的具体参数组合。

### 6.3 命令执行规则

- 禁止把用户输入拼接进 shell 命令字符串。
- 必须以 argv 数组形式执行命令。
- 执行前校验协议、端口和操作类型。
- 设置命令执行超时。
- 捕获 stdout、stderr、exit code 和耗时，用于审计日志。
- 日志中必须隐藏敏感值。

## 7. 认证设计

### 7.1 账号模型

第一版支持一个或多个预配置管理员用户。用户通过配置文件或安装命令在系统外部创建，不提供公开注册功能。

配置示例：

```yaml
auth:
  users:
    - username: admin
      password_hash: "$argon2id$v=19$..."
  session_secret_file: /etc/firewall-manager/session-secret
```

### 7.2 密码存储

- 只保存密码哈希。
- 优先使用 Argon2id；不可用时可使用 bcrypt。
- 禁止保存明文密码。
- 后续提供管理员 CLI，用于重置或轮换密码。

### 7.3 会话处理

- 用户通过用户名和密码登录。
- 登录成功后创建 HTTP-only、same-site 的 session cookie；TLS 启用时 cookie 必须设置 `Secure`。
- 会话内容包含用户标识和过期时间。
- 默认会话有效期：8 小时。
- 退出登录时清除 session cookie。
- 所有 `/api/*` 防火墙管理接口都必须要求已认证会话。

### 7.4 登录流程

```text
GET /login
  -> 渲染登录页

POST /api/auth/login
  -> 校验用户名和密码
  -> 创建会话
  -> 返回成功

GET /app
  -> 要求已登录
  -> 返回前端应用，由 React 渲染管理控制台

POST /api/auth/logout
  -> 清除会话
```

### 7.5 安全控制

- 对登录失败按 IP 和用户名限流。
- 对 cookie 认证的变更类 API 启用 CSRF 防护。
- session cookie 始终设置 `HttpOnly` 和 `SameSite=Strict` 或 `SameSite=Lax`。
- TLS/HTTPS 模式下 session cookie 设置 `Secure=true`；HTTP 模式下不能设置 `Secure=true`。
- 生产环境推荐通过 HTTPS 访问。应用自身终止 TLS，不依赖 Nginx/Caddy。
- 登录失败时返回通用错误，不暴露用户名是否存在或密码是否错误。

## 8. 国际化设计

### 8.1 支持语言

- `zh-CN`：简体中文。
- `en-US`：英文。

### 8.2 语言选择优先级

1. 用户显式选择，并存储在 locale cookie 中。
2. 浏览器 `Accept-Language` 请求头。
3. 配置文件中的默认语言。
4. 兜底语言：`en-US`。

### 8.3 翻译文件结构

```text
locales/
  en-US.json
  zh-CN.json
```

英文示例：

```json
{
  "login.title": "Sign in",
  "login.username": "Username",
  "login.password": "Password",
  "firewall.title": "Firewall",
  "firewall.status.enabled": "Enabled",
  "firewall.status.disabled": "Disabled",
  "ports.open": "Open port",
  "ports.close": "Close port"
}
```

中文示例：

```json
{
  "login.title": "登录",
  "login.username": "用户名",
  "login.password": "密码",
  "firewall.title": "防火墙",
  "firewall.status.enabled": "已启用",
  "firewall.status.disabled": "已禁用",
  "ports.open": "打开端口",
  "ports.close": "关闭端口"
}
```

### 8.4 API 错误国际化

API 错误返回稳定的机器可读错误码，由前端负责翻译展示。

示例：

```json
{
  "error": {
    "code": "PORT_ALREADY_OPEN",
    "message": "Port is already open"
  }
}
```

`message` 只作为兜底信息。UI 展示应优先使用 `code` 映射本地化文案。

## 9. 防火墙领域模型

### 9.1 核心概念

- 防火墙状态：防火墙服务是否启用、是否正在运行。
- 端口规则：防火墙中放行的协议和端口组合。
- 协议：`tcp` 或 `udp`。
- 目标操作：打开端口或关闭端口。
- 系统适配器：负责读取和修改具体操作系统防火墙状态的实现。

### 9.2 数据类型

```text
FirewallState
  osType: "centos" | "ubuntu"
  backend: "firewalld" | "ufw"
  serviceEnabled: boolean
  serviceRunning: boolean
  defaultIncomingPolicy?: "allow" | "deny" | "reject" | "unknown"
  openPorts: PortRule[]
  loadedAt: ISO timestamp

PortRule
  port: number
  protocol: "tcp" | "udp"
  source?: string
  description?: string

PortChangeRequest
  port: number
  protocol: "tcp" | "udp"
```

第一版只管理所有来源的 `tcp` 和 `udp` 端口规则。如果底层防火墙存在复杂规则，只有在能够安全解析时才作为只读信息展示。

## 10. 防火墙服务抽象

### 10.1 接口与通用组合结构

Go 实现中使用一个共享接口定义防火墙操作：

```text
type FirewallService interface {
  Detect(ctx): bool
  LoadState(ctx): FirewallState
  OpenPort(ctx, request): FirewallState
  ClosePort(ctx, request): FirewallState
}
```

通用逻辑通过组合复用：

```text
BaseFirewallService
  commandRunner
  validatePortChange(request)
  normalizeCommandError(error)

CentOSFirewallService
  base: BaseFirewallService

UbuntuFirewallService
  base: BaseFirewallService
```

职责：

- 执行通用输入校验。
- 定义统一返回类型。
- 将操作系统特定命令输出归一化为 `FirewallState`。
- 提供一致的错误模型。

### 10.2 CentOS 实现

类名：`CentOSFirewallService`。

预期后端：通过 `firewall-cmd` 操作 `firewalld`。

检测逻辑：

- 检查 `/etc/os-release` 中的 `ID=centos`。如果希望兼容同族发行版，可同时支持 `ID=rhel`、`ID=rocky`、`ID=almalinux`。
- 检查 `firewall-cmd` 是否存在。
- 检查 `systemctl is-enabled firewalld` 和 `systemctl is-active firewalld`。

加载状态命令：

```text
firewall-cmd --state
firewall-cmd --get-default-zone
firewall-cmd --zone=<zone> --list-ports
systemctl is-enabled firewalld
systemctl is-active firewalld
```

打开端口：

```text
firewall-cmd --zone=<zone> --add-port=<port>/<protocol>
firewall-cmd --permanent --zone=<zone> --add-port=<port>/<protocol>
```

关闭端口：

```text
firewall-cmd --zone=<zone> --remove-port=<port>/<protocol>
firewall-cmd --permanent --zone=<zone> --remove-port=<port>/<protocol>
```

说明：

- 默认 zone 通过 `firewall-cmd --get-default-zone` 加载，除非配置文件显式指定。
- 端口变更必须分别修改 runtime 配置和 permanent 配置。
- runtime 配置用于立即生效，permanent 配置用于重启 firewalld 或系统后保留。
- 禁止在端口开关流程中使用 `firewall-cmd --reload`。
- 禁止使用 `firewall-cmd --runtime-to-permanent`，避免把其他临时 runtime 规则整体固化到 permanent 配置。
- 端口变更后重新调用状态加载命令，并返回最新 `FirewallState`。
- `FirewallD is not running` 应映射为独立错误码。

失败处理：

- 打开端口时，先执行 runtime add，再执行 permanent add。
- 如果 runtime add 成功但 permanent add 失败，尝试执行 runtime remove 回滚本次 runtime 变更，然后返回 `PORT_OPEN_FAILED`。
- 关闭端口时，先执行 runtime remove，再执行 permanent remove。
- 如果 runtime remove 成功但 permanent remove 失败，尝试执行 runtime add 回滚本次 runtime 变更，然后返回 `PORT_CLOSE_FAILED`。
- 回滚失败时必须记录审计日志，并在错误响应中标记为部分失败，提示管理员人工检查 firewalld 状态。

### 10.3 Ubuntu 实现

类名：`UbuntuFirewallService`。

预期后端：`ufw`。

检测逻辑：

- 检查 `/etc/os-release` 中的 `ID=ubuntu`。
- 检查 `ufw` 是否存在。
- 检查 `ufw status`。

加载状态命令：

```text
ufw status verbose
systemctl is-enabled ufw
systemctl is-active ufw
```

打开端口：

```text
ufw allow <port>/<protocol>
```

关闭端口：

```text
ufw delete allow <port>/<protocol>
```

说明：

- 将 `ufw status verbose` 解析为 `FirewallState`。
- UFW 未激活时，`serviceRunning=false`。
- 第一版不要自动启用 UFW。启用防火墙可能影响服务器连接，应作为后续独立功能显式实现。

### 10.4 工厂选择

通过小型工厂选择具体实现：

```text
FirewallServiceFactory.create()
  -> 如果检测到 CentOS/firewalld，返回 CentOSFirewallService
  -> 如果检测到 Ubuntu/ufw，返回 UbuntuFirewallService
  -> 否则抛出 UnsupportedFirewallError
```

应用启动时创建一次选中的服务实例并复用。第一版不需要支持运行期间切换操作系统或防火墙后端。

## 11. 初始系统状态加载

### 11.1 firewalld 变更约束

CentOS/firewalld 的端口开关流程必须遵守以下约束：

- 不能使用 `firewall-cmd --reload`。
- 不能使用 `firewall-cmd --complete-reload`。
- 不能使用 `firewall-cmd --runtime-to-permanent`。
- 只能对用户指定的目标端口执行精确的 runtime 和 permanent 变更。

原因：

- `--reload` 会用 permanent 配置重新生成 runtime 配置，未写入 permanent 的 runtime-only 规则会丢失。
- `--complete-reload` 影响更大，可能中断已有连接，不适合管理网站的端口开关操作。
- `--runtime-to-permanent` 会把当前全部 runtime 配置写入 permanent，可能把管理员临时添加的测试规则或应急规则永久化。
- 管理网站的单次操作应该只影响用户指定的一个端口，便于预测、审计和回滚。

正确策略：

```text
打开端口：runtime add -> permanent add -> query state
关闭端口：runtime remove -> permanent remove -> query state
```

这里的 `query state` 指重新查询系统状态，不是执行 `firewall-cmd --reload`。

### 11.2 启动时加载

应用启动时执行：

1. 检测操作系统和防火墙后端。
2. 创建对应的 `FirewallService` 实例。
3. 加载初始 `FirewallState`。
4. 将状态保存在内存中，作为最近一次已知状态。
5. 记录检测和状态加载日志。

如果防火墙服务只是禁用或未运行，启动不应直接失败。只有操作系统/后端不支持，或必要命令缺失时，启动才应失败。

### 11.3 登录后加载

认证成功进入控制台后，前端调用 `GET /api/firewall/state` 获取最新状态。这样可以避免展示启动时缓存的过期状态。

### 11.4 变更后刷新

打开或关闭端口后，后端重新从系统加载防火墙状态，并在响应中返回更新后的状态。

## 12. HTTP API 设计

所有防火墙 API 都要求认证。

### 12.1 认证接口

```text
POST /api/auth/login
Request:
  { "username": "admin", "password": "..." }
Response:
  { "ok": true }

POST /api/auth/logout
Response:
  { "ok": true }

GET /api/auth/me
Response:
  { "username": "admin" }
```

### 12.2 防火墙状态接口

```text
GET /api/firewall/state
Response:
  {
    "osType": "ubuntu",
    "backend": "ufw",
    "serviceEnabled": true,
    "serviceRunning": true,
    "defaultIncomingPolicy": "deny",
    "openPorts": [
      { "port": 22, "protocol": "tcp" },
      { "port": 443, "protocol": "tcp" }
    ],
    "loadedAt": "2026-07-07T10:00:00Z"
  }
```

### 12.3 打开端口

```text
POST /api/firewall/ports
Request:
  { "port": 443, "protocol": "tcp" }
Response:
  { "state": FirewallState }
```

### 12.4 关闭端口

```text
DELETE /api/firewall/ports/{protocol}/{port}
Response:
  { "state": FirewallState }
```

### 12.5 语言接口

```text
GET /api/locale
Response:
  { "locale": "zh-CN", "supportedLocales": ["zh-CN", "en-US"] }

POST /api/locale
Request:
  { "locale": "en-US" }
Response:
  { "locale": "en-US" }
```

### 12.6 运行时配置接口

```text
GET /api/runtime
Response:
  {
    "tlsEnabled": true,
    "publicUrl": "https://firewall.example.com:10240",
    "allowInsecureRemote": false,
    "version": "0.1.0"
  }
```

用途：

- 前端判断是否展示 HTTPS 安全提示。
- 前端展示当前访问地址和版本信息。
- 前端避免硬编码 TLS/HTTP 行为。

## 13. 校验规则

### 13.1 端口

- 必须是整数。
- 范围：`1` 到 `65535`。
- 转换前拒绝带 shell 元字符或非数字内容的字符串。

### 13.2 协议

- 允许值：`tcp`、`udp`。
- 可以将输入统一转换为小写。

### 13.3 重复操作

- 打开已经打开的端口，可以返回当前状态并视为成功，也可以返回 `PORT_ALREADY_OPEN` 和 409。为简化 UI，第一版建议采用幂等成功。
- 关闭已经关闭的端口，可以返回当前状态并视为成功，也可以返回 `PORT_NOT_OPEN` 和 404。为简化 UI，第一版建议采用幂等成功。

## 14. 错误模型

使用稳定错误码，便于前端翻译和排查问题。

```text
AUTH_INVALID_CREDENTIALS
AUTH_REQUIRED
CSRF_INVALID
UNSUPPORTED_OS
FIREWALL_COMMAND_MISSING
FIREWALL_SERVICE_INACTIVE
FIREWALL_STATE_LOAD_FAILED
PORT_INVALID
PROTOCOL_INVALID
PORT_OPEN_FAILED
PORT_CLOSE_FAILED
COMMAND_TIMEOUT
COMMAND_DENIED
INTERNAL_ERROR
```

HTTP 状态码建议：

- `400`：输入非法。
- `401`：未认证。
- `403`：已认证但无权限、CSRF 失败或命令被拒绝。
- `409`：如果不采用幂等策略，当前状态与操作冲突。
- `500`：未预期的内部错误。
- `503`：防火墙后端不可用或未运行。

## 15. UI 设计

### 15.1 设计原则

- 操作顺序先展示系统状态，再允许修改端口，避免用户在不了解当前状态时直接操作。
- 打开端口是新增暴露面，表单提交前必须展示协议和端口的明确预览。
- 关闭端口可能影响服务可用性，必须二次确认。
- 所有变更都以服务端返回的最新 `FirewallState` 为准，不做乐观更新。
- UI 中所有错误都使用稳定错误码映射本地化文案。
- 桌面端以信息密度和操作效率为主，移动端以卡片和单列流程为主。

### 15.2 路由

前端路由：

```text
/login      登录页
/app        防火墙控制台
/app/ports 端口管理页，第一版可与 /app 使用同一页面
/*          未匹配路径，已登录则回到 /app，未登录则回到 /login
```

后端静态资源 fallback：

- 非 `/api/*` 请求返回 React 应用的 `index.html`。
- `/api/*` 请求只进入后端 API，不走前端 fallback。

### 15.3 首次访问顺序

```text
1. 用户访问 /。
2. 前端调用 GET /api/auth/me。
3. 如果返回 401，跳转 /login。
4. 如果已登录，跳转 /app。
5. /app 加载后调用 GET /api/firewall/state。
6. 成功后展示系统摘要、端口列表和操作区。
7. 失败后展示错误状态和重试按钮。
```

### 15.4 登录页布局

桌面端布局：

```text
┌───────────────────────────────────────────────┐
│ 左侧品牌区                                     │
│ - Firewall Manager                            │
│ - 当前服务器防火墙管理说明                     │
│ - 安全提示：请仅在可信网络访问                 │
├───────────────────────┬───────────────────────┤
│                       │ 登录卡片               │
│                       │ - 语言切换             │
│                       │ - 用户名               │
│                       │ - 密码                 │
│                       │ - 登录按钮             │
│                       │ - 错误提示             │
└───────────────────────┴───────────────────────┘
```

移动端布局：

```text
顶部品牌标题
登录卡片
安全提示
语言切换
```

字段：

- 用户名。
- 密码。
- 语言切换器。

登录响应：

- 初始状态：登录按钮可点击。
- 提交中：按钮显示加载状态，用户名和密码输入框禁用。
- 成功：跳转 `/app`，不在前端存储密码或 token。
- 失败：展示通用错误，例如“用户名或密码错误”。
- 限流：展示“尝试次数过多，请稍后再试”。
- 网络失败：展示“无法连接到服务，请检查网络或服务状态”。

### 15.5 控制台整体布局

桌面端布局：

```text
┌──────────────────────────────────────────────────────────────┐
│ 顶部栏                                                       │
│ Firewall Manager | 状态徽标 | 语言切换 | 当前用户 | 退出登录 │
├──────────────────────────────────────────────────────────────┤
│ 系统摘要卡片区                                               │
│ [OS] [Backend] [服务运行] [默认策略] [最近加载时间]           │
├───────────────────────────────┬──────────────────────────────┤
│ 已开放端口                    │ 打开端口                     │
│ - 搜索/筛选                   │ - 端口输入                   │
│ - 端口表格                    │ - 协议选择                   │
│ - 单项关闭操作                │ - 操作预览                   │
│                               │ - 提交按钮                   │
├───────────────────────────────┴──────────────────────────────┤
│ 最近操作/错误提示区域                                        │
└──────────────────────────────────────────────────────────────┘
```

移动端布局：

```text
顶部栏
系统摘要卡片，横向滚动或单列
打开端口卡片
已开放端口卡片列表
最近操作/错误提示
```

顶部栏内容：

- 产品名称。
- 防火墙运行状态徽标：`Running`、`Inactive`、`Unsupported`。
- 语言切换器。
- 当前用户名。
- 退出登录按钮。

系统摘要卡片：

- 操作系统：`centos` 或 `ubuntu`。
- 防火墙后端：`firewalld` 或 `ufw`。
- 服务是否启用。
- 服务是否运行。
- 默认入站策略。
- 最近加载时间。
- 手动刷新按钮。

端口列表：

- 列：端口、协议、来源、描述、操作。
- 第一版来源固定为所有来源，可显示为 `0.0.0.0/0` 或 `Any`。
- 支持按端口号搜索。
- 支持按协议筛选：全部、TCP、UDP。
- 空状态展示“当前没有由系统识别的开放端口”。

打开端口表单：

- 端口输入框。
- 协议选择：`tcp`、`udp`。
- 操作预览：例如“将打开 TCP 443”。
- 提交按钮：`打开端口`。
- 提示文案：打开端口会使服务可被外部访问，请确认对应服务已正确配置。

### 15.6 初始状态加载

控制台加载顺序：

```text
1. React 应用进入 /app。
2. 调用 GET /api/auth/me。
3. 未登录则跳转 /login。
4. 已登录则调用 GET /api/firewall/state。
5. 请求期间展示骨架屏，不展示空表格。
6. 成功后渲染完整控制台。
7. 失败后展示错误卡片和重试按钮。
```

加载响应：

- `200`：正常渲染状态。
- `401`：跳转 `/login`。
- `503 FIREWALL_SERVICE_INACTIVE`：展示防火墙未运行，禁用打开/关闭按钮。
- `503 FIREWALL_COMMAND_MISSING`：展示缺少系统命令，禁用操作。
- `500 FIREWALL_STATE_LOAD_FAILED`：展示加载失败和重试按钮。

### 15.7 打开端口操作流程

```text
1. 用户输入端口，例如 443。
2. 用户选择协议，例如 tcp。
3. 前端实时校验端口范围和协议。
4. UI 展示操作预览：“将打开 TCP 443”。
5. 用户点击“打开端口”。
6. 前端禁用表单和提交按钮。
7. 前端发送 POST /api/firewall/ports。
8. 后端执行 runtime add -> permanent add -> 重新查询状态。
9. 前端用响应中的 state 替换当前状态。
10. 显示成功提示：“TCP 443 已打开”。
```

打开端口响应：

- 成功：清空端口输入，保留协议选择，刷新端口列表。
- 端口已开放：按幂等成功处理，提示“端口已处于开放状态”。
- 输入非法：字段下方展示错误，不发送请求或展示后端错误。
- 命令失败：展示错误卡片，保留用户输入，允许重试。
- 部分失败：展示高优先级警告，提示管理员人工检查系统防火墙状态。

### 15.8 关闭端口操作流程

```text
1. 用户在端口列表点击“关闭”。
2. 打开确认弹窗。
3. 弹窗展示端口、协议和影响提示。
4. 用户确认关闭。
5. 前端禁用该行按钮。
6. 前端发送 DELETE /api/firewall/ports/{protocol}/{port}。
7. 后端执行 runtime remove -> permanent remove -> 重新查询状态。
8. 前端用响应中的 state 替换当前状态。
9. 显示成功提示：“TCP 443 已关闭”。
```

确认弹窗文案要求：

- 标题：`确认关闭端口`。
- 内容：`关闭 TCP 443 可能导致依赖该端口的服务无法从外部访问。`。
- 主按钮：`确认关闭`。
- 次按钮：`取消`。

关闭端口响应：

- 成功：端口从列表消失或标记为已关闭，然后由最新 state 驱动渲染。
- 端口本来未开放：按幂等成功处理，刷新状态。
- 命令失败：该行恢复可操作状态，展示错误卡片。
- 部分失败：展示高优先级警告，提示人工检查。

### 15.9 手动刷新

刷新按钮位于系统摘要卡片右上角。

流程：

```text
1. 用户点击刷新。
2. 前端调用 GET /api/firewall/state。
3. 请求期间只禁用刷新按钮，不阻塞整个页面。
4. 成功后替换全局 state。
5. 失败后保留旧 state，同时展示“刷新失败”。
```

### 15.10 退出登录

流程：

```text
1. 用户点击退出登录。
2. 前端调用 POST /api/auth/logout。
3. 无论请求成功或 session 已失效，都清理前端内存状态。
4. 跳转 /login。
```

### 15.11 全局响应状态

全局响应状态包括：

- Loading：骨架屏或按钮 spinner。
- Success：右下角 toast，3 到 5 秒自动消失。
- Warning：页面内警告卡片，需要用户手动关闭。
- Error：页面内错误卡片，附带重试按钮或排查建议。
- Disabled：当防火墙不可用、命令缺失或未认证时禁用操作。

错误展示顺序：

1. 字段级错误，例如端口非法。
2. 操作级错误，例如打开端口失败。
3. 页面级错误，例如防火墙状态加载失败。
4. 系统级错误，例如不支持的操作系统。

### 15.12 可访问性和移动端

- 所有按钮必须有明确文本，不只依赖图标。
- 表单字段必须有 label。
- Toast 不能作为唯一错误反馈，重要错误必须出现在页面中。
- 确认弹窗支持键盘操作和 Escape 关闭。
- 移动端端口列表使用卡片，避免横向表格难读。
- 小屏幕上打开端口表单放在端口列表之前，减少用户查找操作入口的成本。

## 16. 配置设计

示例：

```yaml
server:
  host: 0.0.0.0
  port: 10240
  public_url: https://firewall.example.com:10240
  tls:
    enabled: true
    cert_file: /etc/firewall-manager/tls.crt
    key_file: /etc/firewall-manager/tls.key

auth:
  users:
    - username: admin
      password_hash: "$argon2id$v=19$..."
  session_secret_file: /etc/firewall-manager/session-secret
  session_ttl_minutes: 480

i18n:
  default_locale: zh-CN
  supported_locales:
    - zh-CN
    - en-US

firewall:
  os: auto
  centos:
    zone: auto
  command_timeout_seconds: 10
```

说明：

- 默认监听高位端口，避免低权限进程绑定 `80` 或 `443`。
- 不使用 Nginx/Caddy 时，如果需要远程访问，建议启用 Go 内置 TLS。
- 一键部署脚本可以生成自签名证书，也可以使用用户提供的证书文件。
- 如果只允许本机访问，可将 `host` 改为 `127.0.0.1` 并关闭 TLS。

### 16.1 TLS/HTTPS 模式

启用 TLS 时：

```yaml
server:
  host: 0.0.0.0
  port: 10240
  public_url: https://firewall.example.com:10240
  tls:
    enabled: true
    cert_file: /etc/firewall-manager/tls.crt
    key_file: /etc/firewall-manager/tls.key
```

行为：

- Go 进程直接提供 HTTPS。
- 登录页、前端静态资源和 `/api/*` 都通过同一个 HTTPS 入口访问。
- session cookie 必须设置 `Secure=true`。
- session cookie 设置 `HttpOnly=true`。
- session cookie 设置 `SameSite=Strict`，如果发现浏览器场景兼容性问题再降为 `Lax`。
- CSRF cookie 如果需要被前端读取，应设置 `HttpOnly=false`，但必须只存随机 token，不存敏感信息。
- `public_url` 用于安装完成后的提示、回调地址生成和前端运行时显示。

证书来源：

- 用户提供正式证书：推荐生产使用。
- 一键部署生成自签名证书：适合内网或测试环境，浏览器会提示证书不受信任。
- 后续可扩展 ACME 自动证书，但第一版不实现，避免引入额外后台逻辑。

### 16.2 非 TLS/HTTP 模式

关闭 TLS 时：

```yaml
server:
  host: 127.0.0.1
  port: 10240
  public_url: http://127.0.0.1:10240
  tls:
    enabled: false
```

行为：

- Go 进程提供普通 HTTP。
- session cookie 不能设置 `Secure=true`，否则浏览器不会在 HTTP 请求中发送 cookie。
- session cookie 仍必须设置 `HttpOnly=true`。
- session cookie 仍建议设置 `SameSite=Strict`。
- 前端必须在登录页和控制台顶部展示“当前未启用 HTTPS”的安全提示。

使用限制：

- HTTP 模式仅建议用于本机访问、受信任内网或开发环境。
- 如果 `host` 不是 `127.0.0.1` 且 TLS 关闭，启动时应打印明显警告日志。
- 如果配置 `allow_insecure_remote=false`，则禁止在 `0.0.0.0` 或非 loopback 地址上关闭 TLS 启动。

推荐配置：

```yaml
server:
  host: 127.0.0.1
  port: 10240
  public_url: http://127.0.0.1:10240
  tls:
    enabled: false
  allow_insecure_remote: false
```

### 16.3 前端运行时安全提示

前端启动后根据 `GET /api/runtime` 或初始配置接口获取运行模式：

```json
{
  "tlsEnabled": true,
  "publicUrl": "https://firewall.example.com:10240",
  "allowInsecureRemote": false
}
```

UI 行为：

- `tlsEnabled=true`：顶部不展示安全警告。
- `tlsEnabled=false` 且当前访问来源是 `localhost` 或 `127.0.0.1`：展示低优先级提示“当前为本机 HTTP 模式”。
- `tlsEnabled=false` 且当前访问来源不是 loopback：展示高优先级警告“当前未启用 HTTPS，请仅在可信网络使用”。
- 登录页也必须展示相同级别的 HTTPS 状态提示。

### 16.4 Cookie 配置矩阵

| 访问模式 | Cookie Secure | HttpOnly | SameSite | 备注 |
| --- | --- | --- | --- | --- |
| HTTPS | true | true | Strict | 推荐生产模式 |
| HTTP loopback | false | true | Strict | 本机或开发模式 |
| HTTP remote | false | true | Strict | 不推荐，默认应禁止或强警告 |

CSRF token 如果通过 cookie 暴露给前端读取，则该 CSRF cookie 不能设置 `HttpOnly`；session cookie 必须始终 `HttpOnly`。

## 17. 日志与审计

需要记录的事件：

- 应用启动和选中的防火墙后端。
- 认证成功和失败，不记录密码。
- 防火墙状态加载成功和失败。
- 打开/关闭端口请求。
- 命令执行结果：命令名、脱敏参数、退出码、耗时。

审计日志示例：

```json
{
  "timestamp": "2026-07-07T10:00:00Z",
  "user": "admin",
  "action": "open_port",
  "port": 443,
  "protocol": "tcp",
  "result": "success"
}
```

## 18. 测试策略

### 18.1 单元测试

- 端口和协议校验。
- 语言选择逻辑。
- 密码哈希校验。
- 防火墙命令构造。
- `firewall-cmd --list-ports` 解析器。
- `ufw status verbose` 解析器。
- 命令失败到错误码的映射。

### 18.2 集成测试

- 登录成功和失败。
- 防火墙 API 的认证保护。
- 使用 mock command runner 测试 `GET /api/firewall/state`。
- 使用 mock command runner 测试打开/关闭端口。
- 中文和英文 UI 文案加载。

### 18.3 手工系统测试

Ubuntu：

- 以低权限用户安装应用。
- 确认进程不是 root。
- 使用配置的管理员账号登录。
- 加载 UFW 初始状态。
- 打开和关闭一个测试 TCP 端口。
- 确认 `ufw status` 反映了变更。

CentOS：

- 以低权限用户安装应用。
- 确认进程不是 root。
- 使用配置的管理员账号登录。
- 加载 firewalld 初始状态。
- 打开和关闭一个测试 TCP 端口。
- 确认 `firewall-cmd --list-ports` 反映了变更。

## 19. 部署说明

### 19.1 部署目标

生产环境部署目标：

- 一个 `firewall-manager` Go 二进制。
- 一个 systemd service。
- 一个低权限系统用户。
- 一个配置目录：`/etc/firewall-manager`。
- 一个数据目录：`/var/lib/firewall-manager`。
- 一个日志目录：`/var/log/firewall-manager`。
- 一个按后端选择的 sudoers 文件，或 root-owned helper。
- 不安装、不启动 Nginx/Caddy。
- 不在生产环境运行 Node.js、Vite dev server 或其他前端进程。

### 19.2 前端资源发布方式

推荐方式：构建时将 `frontend/dist` 通过 Go `embed` 打进 `firewall-manager` 二进制。

优点：

- 一键部署只需要安装一个二进制。
- 前端和后端版本天然一致。
- 不需要维护 `/usr/share/firewall-manager/web` 文件同步。

备选方式：一键部署时将 `frontend/dist` 安装到：

```text
/usr/share/firewall-manager/web
```

后端进程读取该目录并托管静态文件。该方式方便调试和替换静态资源，但发布一致性弱于 `embed`。

### 19.3 systemd Service

服务应以专用低权限用户运行：

```ini
[Unit]
Description=Firewall Manager
After=network-online.target

[Service]
User=firewall-manager
Group=firewall-manager
Environment=FIREWALL_MANAGER_CONFIG=/etc/firewall-manager/config.yml
ExecStart=/usr/local/bin/firewall-manager
Restart=on-failure
PrivateTmp=true
ProtectHome=true
ProtectSystem=strict
ReadWritePaths=/var/lib/firewall-manager /var/log/firewall-manager

[Install]
WantedBy=multi-user.target
```

如果第一版直接从 Web 进程使用 sudoers 提权，不设置 `NoNewPrivileges=true`，否则可能导致 sudo 无法提权。如果改为 root-owned helper，可重新启用 `NoNewPrivileges=true`。

### 19.4 一键部署脚本

提供安装脚本：

```text
scripts/install.sh
```

脚本职责：

1. 检查当前系统是 Ubuntu 或 CentOS。
2. 检查必要命令：`systemctl`、`sudo`、`firewall-cmd` 或 `ufw`。
3. 创建低权限用户和组：`firewall-manager`。
4. 创建目录：`/etc/firewall-manager`、`/var/lib/firewall-manager`、`/var/log/firewall-manager`。
5. 安装 `firewall-manager` 二进制到 `/usr/local/bin/firewall-manager`。
6. 如果使用外置静态资源，将 `frontend/dist` 安装到 `/usr/share/firewall-manager/web`。
7. 生成或写入 `/etc/firewall-manager/config.yml`。
8. 生成 session secret。
9. 生成管理员初始密码哈希，或提示用户传入 `--admin-password`。
10. 写入 systemd service 文件。
11. 按系统自动匹配 UFW 或 firewalld，并写入 `sudoers.ufw` 或 `sudoers.firewalld`。
12. 执行 `systemctl daemon-reload`。
13. 启用并启动服务：`systemctl enable --now firewall-manager`。
14. 输出访问地址和初始管理员信息。

脚本参数建议：

```text
--listen-host 0.0.0.0
--listen-port 10240
--admin-user admin
--admin-password <password>
--tls-cert /path/to/cert.pem
--tls-key /path/to/key.pem
--generate-self-signed-cert
--no-tls
--allow-insecure-remote
--firewall-zone public
--no-sudo
```

### 19.5 构建流程

发布包构建流程：

```text
1. 在构建机安装 Node.js 和 Go。
2. frontend/ 执行 npm ci。
3. frontend/ 执行 npm run build，生成 dist。
4. backend/ 通过 go build 构建 firewall-manager。
5. 如果采用 embed，go build 时将 frontend/dist 打进二进制。
6. 产出 release 包：firewall-manager、install.sh、示例配置、UFW/firewalld sudoers 示例。
```

生产服务器执行一键部署时不需要 Node.js。

## 20. 实施里程碑

1. 创建 Go 后端项目骨架，包括 HTTP server、配置加载、结构化日志和测试环境。
2. 创建 Vite React TypeScript Tailwind 前端项目骨架。
3. 实现后端静态资源托管和前端路由 `index.html` fallback。
4. 实现无注册的认证功能、签名 session cookie 和 CSRF 防护。
5. 实现前端 i18n 资源和语言切换。
6. 实现 argv 命令执行器，包含超时、审计日志和测试。
7. 实现 `FirewallService` 接口、`BaseFirewallService` 组合结构和工厂。
8. 实现 Ubuntu `ufw` 服务和解析器测试。
9. 实现 CentOS `firewall-cmd` 服务和解析器测试，禁止 reload/runtime-to-permanent。
10. 实现防火墙状态 API。
11. 实现打开/关闭端口 API。
12. 构建现代控制台 UI 并接入 API。
13. 添加前端 build 与 Go embed 发布流程。
14. 添加 systemd、sudoers/helper 和一键部署脚本。
15. 在 Ubuntu 和 CentOS 主机上验证。

## 21. 待决策问题

- 权限模型：第一版使用 sudoers 白名单，还是一开始就实现 root-owned helper。
- 是否把不支持编辑的复杂防火墙规则以只读方式展示。
- 是否支持 CentOS 之外的 RHEL 兼容发行版。
