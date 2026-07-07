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
		writeError(w, http.StatusBadRequest, "INVALID_JSON")
		return
	}
	if req.Username != h.deps.Config.Auth.AdminUser || req.Password != h.deps.Config.Auth.AdminPassword {
		writeError(w, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS")
		return
	}
	if err := h.deps.Sessions.Set(w, req.Username, h.deps.Config.Server.TLS.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR")
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
		writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": claims.Username})
}

func (h handler) locale(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"locale": "zh-CN", "supportedLocales": []string{"zh-CN", "en-US"}})
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
		writeError(w, http.StatusInternalServerError, firewallErrorCode(err, "FIREWALL_STATE_LOAD_FAILED"))
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (h handler) openPort(w http.ResponseWriter, r *http.Request) {
	var req firewall.PortChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON")
		return
	}
	if err := validatePort(req.Port, req.Protocol); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := h.deps.FirewallService.OpenPort(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, firewallErrorCode(err, "PORT_OPEN_FAILED"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]firewall.State{"state": state})
}

func (h handler) closePort(w http.ResponseWriter, r *http.Request) {
	protocol := r.PathValue("protocol")
	port, err := strconv.Atoi(r.PathValue("port"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "PORT_INVALID")
		return
	}
	if err := validatePort(port, protocol); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	state, err := h.deps.FirewallService.ClosePort(r.Context(), firewall.PortChangeRequest{Port: port, Protocol: protocol})
	if err != nil {
		writeError(w, http.StatusInternalServerError, firewallErrorCode(err, "PORT_CLOSE_FAILED"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]firewall.State{"state": state})
}

func (h handler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.deps.Sessions.Get(r); !ok {
			writeError(w, http.StatusUnauthorized, "AUTH_REQUIRED")
			return
		}
		next(w, r)
	}
}

func validatePort(port int, protocol string) error {
	if port < 1 || port > 65535 {
		return apiError("PORT_INVALID")
	}
	protocol = strings.ToLower(protocol)
	if protocol != "tcp" && protocol != "udp" {
		return apiError("PROTOCOL_INVALID")
	}
	return nil
}

type apiError string

func (e apiError) Error() string { return string(e) }

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": code}})
}

func firewallErrorCode(err error, fallback string) string {
	var fwErr firewall.Error
	if errors.As(err, &fwErr) {
		return fwErr.Code
	}
	return fallback
}
