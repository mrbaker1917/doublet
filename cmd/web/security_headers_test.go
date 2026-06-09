package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersOnResponses(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := securityHeaders(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertHeader(t, rec, "Content-Security-Policy", cspPolicy)
	assertHeader(t, rec, "X-Frame-Options", "DENY")
	assertHeader(t, rec, "X-Content-Type-Options", "nosniff")
	assertHeader(t, rec, "Referrer-Policy", referrerPolicy)
	assertHeader(t, rec, "Permissions-Policy", permissionsPolicy)
	assertHeaderMissing(t, rec, "Strict-Transport-Security")
}

func TestSecurityHeadersSetHSTSOnHTTPS(t *testing.T) {
	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	t.Run("tls", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.TLS = &tls.ConnectionState{}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertHeader(t, rec, "Strict-Transport-Security", strictTransportSecs)
	})

	t.Run("forwarded proto", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assertHeader(t, rec, "Strict-Transport-Security", strictTransportSecs)
	})
}

func assertHeader(t *testing.T, rec *httptest.ResponseRecorder, key, want string) {
	t.Helper()
	got := rec.Header().Get(key)
	if got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}

func assertHeaderMissing(t *testing.T, rec *httptest.ResponseRecorder, key string) {
	t.Helper()
	if got := rec.Header().Get(key); got != "" {
		t.Fatalf("%s = %q, want absent", key, got)
	}
}
