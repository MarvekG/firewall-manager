package staticweb

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterServesIndexForRootAndSPARoute(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux)

	for _, path := range []string{"/", "/app", "/app/ports"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d", path, recorder.Code)
		}
		if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
			t.Fatalf("%s unexpected content type: %s", path, recorder.Header().Get("Content-Type"))
		}
	}
}

func TestRegisterDoesNotFallbackAPI(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/missing", nil)
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}
