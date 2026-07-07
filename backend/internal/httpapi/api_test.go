package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"firewall-manager/backend/internal/auth"
	"firewall-manager/backend/internal/config"
	"firewall-manager/backend/internal/firewall"
)

func TestRuntimeEndpoint(t *testing.T) {
	server := newTestServer()
	response := request(t, server, http.MethodGet, "/api/runtime", nil, nil)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	var body map[string]any
	decode(t, response.Body, &body)
	if body["publicUrl"] != "http://127.0.0.1:10240" || body["tlsEnabled"] != false {
		t.Fatalf("unexpected runtime body: %#v", body)
	}
}

func TestAuthFlow(t *testing.T) {
	server := newTestServer()

	unauthorized := request(t, server, http.MethodGet, "/api/auth/me", nil, nil)
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 before login, got %d", unauthorized.StatusCode)
	}

	login := request(t, server, http.MethodPost, "/api/auth/login", []byte(`{"username":"admin","password":"admin"}`), nil)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("expected login 200, got %d", login.StatusCode)
	}
	cookies := login.Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie")
	}

	me := request(t, server, http.MethodGet, "/api/auth/me", nil, cookies)
	if me.StatusCode != http.StatusOK {
		t.Fatalf("expected me 200, got %d", me.StatusCode)
	}
	var body map[string]string
	decode(t, me.Body, &body)
	if body["username"] != "admin" {
		t.Fatalf("unexpected me body: %#v", body)
	}

	logout := request(t, server, http.MethodPost, "/api/auth/logout", nil, cookies)
	if logout.StatusCode != http.StatusOK {
		t.Fatalf("expected logout 200, got %d", logout.StatusCode)
	}
}

func TestInvalidLogin(t *testing.T) {
	server := newTestServer()
	response := request(t, server, http.MethodPost, "/api/auth/login", []byte(`{"username":"admin","password":"wrong"}`), nil)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.StatusCode)
	}
	assertError(t, response, "AUTH_INVALID_CREDENTIALS", "用户名或密码错误。")
}

func TestErrorMessageUsesLocale(t *testing.T) {
	server := newTestServer()
	response := request(t, server, http.MethodPost, "/api/auth/login", []byte(`{"username":"admin","password":"wrong"}`), []*http.Cookie{{Name: "fm_locale", Value: "en-US"}})
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.StatusCode)
	}
	assertError(t, response, "AUTH_INVALID_CREDENTIALS", "Invalid username or password.")
}

func TestErrorMessageUsesAcceptLanguage(t *testing.T) {
	server := newTestServer()
	response := requestWithHeaders(t, server, http.MethodPost, "/api/auth/login", []byte(`{"username":"admin","password":"wrong"}`), nil, map[string]string{"Accept-Language": "en-US,en;q=0.9"})
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.StatusCode)
	}
	assertError(t, response, "AUTH_INVALID_CREDENTIALS", "Invalid username or password.")
}

func TestFirewallEndpointsRequireAuth(t *testing.T) {
	server := newTestServer()
	response := request(t, server, http.MethodGet, "/api/firewall/state", nil, nil)
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.StatusCode)
	}
	assertErrorCode(t, response, "AUTH_REQUIRED")
}

func TestFirewallPortFlow(t *testing.T) {
	server := newTestServer()
	cookies := loginCookies(t, server)

	state := request(t, server, http.MethodGet, "/api/firewall/state", nil, cookies)
	if state.StatusCode != http.StatusOK {
		t.Fatalf("expected state 200, got %d", state.StatusCode)
	}
	var initial firewall.State
	decode(t, state.Body, &initial)
	if hasPort(initial.OpenPorts, "10240", "tcp") {
		t.Fatalf("expected own listen port to be hidden: %#v", initial.OpenPorts)
	}

	open := request(t, server, http.MethodPost, "/api/firewall/ports", []byte(`{"port":"443,1000-1005","protocol":"tcp"}`), cookies)
	if open.StatusCode != http.StatusOK {
		t.Fatalf("expected open 200, got %d", open.StatusCode)
	}
	var opened struct {
		State firewall.State `json:"state"`
	}
	decode(t, open.Body, &opened)
	if !hasPort(opened.State.OpenPorts, "443", "tcp") {
		t.Fatalf("expected 443/tcp in state: %#v", opened.State.OpenPorts)
	}
	if !hasPort(opened.State.OpenPorts, "1000-1005", "tcp") {
		t.Fatalf("expected 1000-1005/tcp in state: %#v", opened.State.OpenPorts)
	}
	if hasPort(opened.State.OpenPorts, "10240", "tcp") {
		t.Fatalf("expected own listen port to be hidden after open: %#v", opened.State.OpenPorts)
	}

	closeResp := request(t, server, http.MethodDelete, "/api/firewall/ports/tcp/1000-1005", nil, cookies)
	if closeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected close 200, got %d", closeResp.StatusCode)
	}
	var closed struct {
		State firewall.State `json:"state"`
	}
	decode(t, closeResp.Body, &closed)
	if hasPort(closed.State.OpenPorts, "1000-1005", "tcp") {
		t.Fatalf("expected 1000-1005/tcp removed: %#v", closed.State.OpenPorts)
	}
	if !hasPort(closed.State.OpenPorts, "443", "tcp") {
		t.Fatalf("expected 443/tcp to remain: %#v", closed.State.OpenPorts)
	}
}

func TestOpenPortValidation(t *testing.T) {
	server := newTestServer()
	cookies := loginCookies(t, server)

	response := request(t, server, http.MethodPost, "/api/firewall/ports", []byte(`{"port":70000,"protocol":"tcp"}`), cookies)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
	assertErrorCode(t, response, "PORT_INVALID")
}

func TestLocaleEndpoints(t *testing.T) {
	server := newTestServer()
	response := request(t, server, http.MethodGet, "/api/locale", nil, nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected locale 200, got %d", response.StatusCode)
	}

	set := request(t, server, http.MethodPost, "/api/locale", []byte(`{"locale":"en-US"}`), nil)
	if set.StatusCode != http.StatusOK {
		t.Fatalf("expected set locale 200, got %d", set.StatusCode)
	}
	if len(set.Cookies()) != 1 || set.Cookies()[0].Value != "en-US" {
		t.Fatalf("expected locale cookie, got %#v", set.Cookies())
	}

	current := request(t, server, http.MethodGet, "/api/locale", nil, set.Cookies())
	if current.StatusCode != http.StatusOK {
		t.Fatalf("expected current locale 200, got %d", current.StatusCode)
	}
	var body struct {
		Locale string `json:"locale"`
	}
	decode(t, current.Body, &body)
	if body.Locale != "en-US" {
		t.Fatalf("expected locale en-US, got %#v", body)
	}
}

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	Register(mux, Dependencies{
		Config: config.Config{
			Server: config.ServerConfig{Host: "127.0.0.1", Port: 10240, PublicURL: "http://127.0.0.1:10240"},
			Auth:   config.AuthConfig{AdminUser: "admin", AdminPassword: "admin", SessionSecret: "test-secret", SessionTTL: time.Hour},
		},
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		Sessions:        auth.NewSessionManager([]byte("test-secret"), time.Hour),
		FirewallService: newFakeFirewallService(),
	})
	return httptest.NewServer(mux)
}

type fakeFirewallService struct {
	mu    sync.Mutex
	ports []firewall.PortRule
}

func newFakeFirewallService() *fakeFirewallService {
	return &fakeFirewallService{ports: []firewall.PortRule{{Port: "22", Protocol: "tcp", Source: "Any", Description: "SSH"}, {Port: "10240", Protocol: "tcp", Source: "Any", Description: "Firewall Manager"}}}
}

func (s *fakeFirewallService) LoadState(ctx context.Context) (firewall.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state(), nil
}

func (s *fakeFirewallService) OpenPort(ctx context.Context, request firewall.PortChangeRequest) (firewall.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, err := firewall.ValidatePortChange(request)
	if err != nil {
		return firewall.State{}, err
	}
	specs, _ := firewall.ParsePortExpression(req.Port)
	for _, spec := range specs {
		exists := false
		for _, port := range s.ports {
			if port.Port == spec.Value && port.Protocol == req.Protocol {
				exists = true
				break
			}
		}
		if !exists {
			s.ports = append(s.ports, firewall.PortRule{Port: spec.Value, Protocol: req.Protocol, Source: "Any"})
		}
	}
	return s.state(), nil
}

func (s *fakeFirewallService) ClosePort(ctx context.Context, request firewall.PortChangeRequest) (firewall.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	req, err := firewall.ValidatePortChange(request)
	if err != nil {
		return firewall.State{}, err
	}
	specs, _ := firewall.ParsePortExpression(req.Port)
	remove := map[string]bool{}
	for _, spec := range specs {
		remove[spec.Value] = true
	}
	s.ports = slices.DeleteFunc(s.ports, func(port firewall.PortRule) bool {
		return remove[port.Port] && port.Protocol == req.Protocol
	})
	return s.state(), nil
}

func (s *fakeFirewallService) state() firewall.State {
	return firewall.State{
		OSType:                "test",
		Backend:               "fake",
		ServiceEnabled:        true,
		ServiceRunning:        true,
		DefaultIncomingPolicy: "deny",
		OpenPorts:             slices.Clone(s.ports),
		LoadedAt:              time.Now().UTC(),
	}
}

func loginCookies(t *testing.T, server *httptest.Server) []*http.Cookie {
	t.Helper()
	response := request(t, server, http.MethodPost, "/api/auth/login", []byte(`{"username":"admin","password":"admin"}`), nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d", response.StatusCode)
	}
	return response.Cookies()
}

func request(t *testing.T, server *httptest.Server, method, path string, body []byte, cookies []*http.Cookie) *http.Response {
	t.Helper()
	return requestWithHeaders(t, server, method, path, body, cookies, nil)
}

func requestWithHeaders(t *testing.T, server *httptest.Server, method, path string, body []byte, cookies []*http.Cookie, headers map[string]string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	response, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func decode(t *testing.T, body io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func assertErrorCode(t *testing.T, response *http.Response, code string) {
	t.Helper()
	assertError(t, response, code, "")
}

func assertError(t *testing.T, response *http.Response, code string, message string) {
	t.Helper()
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decode(t, response.Body, &body)
	if body.Error.Code != code {
		t.Fatalf("expected error %s, got %#v", code, body)
	}
	if message != "" && body.Error.Message != message {
		t.Fatalf("expected error message %q, got %#v", message, body)
	}
}

func hasPort(ports []firewall.PortRule, port string, protocol string) bool {
	for _, rule := range ports {
		if rule.Port == port && rule.Protocol == protocol {
			return true
		}
	}
	return false
}
