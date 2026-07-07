package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSessionSetGetAndClear(t *testing.T) {
	manager := NewSessionManager([]byte("test-secret"), time.Hour)
	recorder := httptest.NewRecorder()

	if err := manager.Set(recorder, "admin", true); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected one cookie, got %d", len(cookies))
	}
	if !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("unexpected cookie flags: %#v", cookies[0])
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookies[0])
	claims, ok := manager.Get(request)
	if !ok || claims.Username != "admin" {
		t.Fatalf("expected valid admin session, got ok=%v claims=%#v", ok, claims)
	}

	clearRecorder := httptest.NewRecorder()
	manager.Clear(clearRecorder, false)
	cleared := clearRecorder.Result().Cookies()[0]
	if cleared.MaxAge != -1 || cleared.Secure {
		t.Fatalf("unexpected clear cookie: %#v", cleared)
	}
}

func TestSessionRejectsTamperedCookie(t *testing.T) {
	manager := NewSessionManager([]byte("test-secret"), time.Hour)
	recorder := httptest.NewRecorder()
	if err := manager.Set(recorder, "admin", false); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	cookie := recorder.Result().Cookies()[0]
	cookie.Value = strings.Replace(cookie.Value, ".", ".tampered", 1)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookie)
	if _, ok := manager.Get(request); ok {
		t.Fatalf("expected tampered cookie to be rejected")
	}
}

func TestSessionExpires(t *testing.T) {
	manager := NewSessionManager([]byte("test-secret"), -time.Second)
	recorder := httptest.NewRecorder()
	if err := manager.Set(recorder, "admin", false); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(recorder.Result().Cookies()[0])
	if _, ok := manager.Get(request); ok {
		t.Fatalf("expected expired session to be rejected")
	}
}
