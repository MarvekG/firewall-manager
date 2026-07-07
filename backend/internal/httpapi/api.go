package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"firewall-manager/backend/internal/auth"
	"firewall-manager/backend/internal/config"
	"firewall-manager/backend/internal/firewall"
)

type Dependencies struct {
	Config          config.Config
	Logger          *slog.Logger
	Sessions        *auth.SessionManager
	FirewallService firewall.Service
}

func Register(mux *http.ServeMux, deps Dependencies) {
	h := handler{deps: deps}
	mux.HandleFunc("GET /api/runtime", h.runtime)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
	mux.HandleFunc("GET /api/locale", h.locale)
	mux.HandleFunc("POST /api/locale", h.setLocale)
	mux.HandleFunc("GET /api/firewall/state", h.requireAuth(h.firewallState))
	mux.HandleFunc("POST /api/firewall/ports", h.requireAuth(h.openPort))
	mux.HandleFunc("DELETE /api/firewall/ports/{protocol}/{port}", h.requireAuth(h.closePort))
}

type handler struct {
	deps Dependencies
}

func (h handler) runtime(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"tlsEnabled":          h.deps.Config.Server.TLS.Enabled,
		"publicUrl":           h.deps.Config.Server.PublicURL,
		"allowInsecureRemote": h.deps.Config.Server.AllowInsecureRemote,
		"version":             "0.1.0-dev",
	})
}

func (h handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}
	if req.Username != h.deps.Config.Auth.AdminUser || req.Password != h.deps.Config.Auth.AdminPassword {
		writeError(w, r, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS")
		return
	}
	if err := h.deps.Sessions.Set(w, req.Username, h.deps.Config.Server.TLS.Enabled); err != nil {
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h handler) logout(w http.ResponseWriter, r *http.Request) {
	h.deps.Sessions.Clear(w, h.deps.Config.Server.TLS.Enabled)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h handler) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.deps.Sessions.Get(r)
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": claims.Username})
}

func (h handler) locale(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"locale": requestLocale(r), "supportedLocales": []string{"zh-CN", "en-US"}})
}

func (h handler) setLocale(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Locale string `json:"locale"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Locale != "en-US" {
		req.Locale = "zh-CN"
	}
	http.SetCookie(w, &http.Cookie{Name: "fm_locale", Value: req.Locale, Path: "/", SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]string{"locale": req.Locale})
}

func (h handler) firewallState(w http.ResponseWriter, r *http.Request) {
	state, err := h.deps.FirewallService.LoadState(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, firewallErrorCode(err, "FIREWALL_STATE_LOAD_FAILED"))
		return
	}
	writeJSON(w, http.StatusOK, h.frontendState(state))
}

func (h handler) openPort(w http.ResponseWriter, r *http.Request) {
	var req firewall.PortChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON")
		return
	}
	validated, err := firewall.ValidatePortChange(req)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, firewallErrorCode(err, "PORT_INVALID"))
		return
	}
	state, err := h.deps.FirewallService.OpenPort(r.Context(), validated)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, firewallErrorCode(err, "PORT_OPEN_FAILED"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]firewall.State{"state": h.frontendState(state)})
}

func (h handler) closePort(w http.ResponseWriter, r *http.Request) {
	protocol := r.PathValue("protocol")
	port := r.PathValue("port")
	validated, err := firewall.ValidatePortChange(firewall.PortChangeRequest{Port: port, Protocol: protocol})
	if err != nil {
		writeError(w, r, http.StatusBadRequest, firewallErrorCode(err, "PORT_INVALID"))
		return
	}
	state, err := h.deps.FirewallService.ClosePort(r.Context(), validated)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, firewallErrorCode(err, "PORT_CLOSE_FAILED"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]firewall.State{"state": h.frontendState(state)})
}

func (h handler) frontendState(state firewall.State) firewall.State {
	ownPort := strconv.Itoa(h.deps.Config.Server.Port)
	ports := make([]firewall.PortRule, 0, len(state.OpenPorts))
	for _, rule := range state.OpenPorts {
		if rule.Port == ownPort {
			continue
		}
		ports = append(ports, rule)
	}
	state.OpenPorts = ports
	return state
}

func (h handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.deps.Sessions.Get(r); !ok {
			writeError(w, r, http.StatusUnauthorized, "AUTH_REQUIRED")
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": localizedErrorMessage(code, requestLocale(r))}})
}

func firewallErrorCode(err error, fallback string) string {
	var fwErr firewall.Error
	if errors.As(err, &fwErr) {
		return fwErr.Code
	}
	return fallback
}

func requestLocale(r *http.Request) string {
	if cookie, err := r.Cookie("fm_locale"); err == nil && cookie.Value == "en-US" {
		return "en-US"
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Accept-Language")), "en") {
		return "en-US"
	}
	return "zh-CN"
}

func localizedErrorMessage(code string, locale string) string {
	messages := map[string]map[string]string{
		"zh-CN": {
			"INVALID_JSON":               "请求数据格式无效。",
			"AUTH_INVALID_CREDENTIALS":   "用户名或密码错误。",
			"AUTH_REQUIRED":              "请先登录。",
			"INTERNAL_ERROR":             "服务器内部错误。",
			"FIREWALL_STATE_LOAD_FAILED": "无法读取防火墙状态。",
			"PORT_INVALID":               "端口或协议无效。",
			"PROTOCOL_INVALID":           "协议必须是 TCP 或 UDP。",
			"PORT_OPEN_FAILED":           "打开端口失败。",
			"PORT_CLOSE_FAILED":          "关闭端口失败。",
		},
		"en-US": {
			"INVALID_JSON":               "Invalid request data.",
			"AUTH_INVALID_CREDENTIALS":   "Invalid username or password.",
			"AUTH_REQUIRED":              "Please sign in first.",
			"INTERNAL_ERROR":             "Internal server error.",
			"FIREWALL_STATE_LOAD_FAILED": "Failed to load firewall state.",
			"PORT_INVALID":               "Invalid port or protocol.",
			"PROTOCOL_INVALID":           "Protocol must be TCP or UDP.",
			"PORT_OPEN_FAILED":           "Failed to open port.",
			"PORT_CLOSE_FAILED":          "Failed to close port.",
		},
	}
	if value := messages[locale][code]; value != "" {
		return value
	}
	if value := messages["zh-CN"][code]; value != "" {
		return value
	}
	return code
}
