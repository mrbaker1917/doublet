package main

import (
	"net/http"
	"strings"
)

const (
	cspPolicy           = "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; font-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"
	permissionsPolicy   = "camera=(), microphone=(), geolocation=(), payment=()"
	referrerPolicy      = "strict-origin-when-cross-origin"
	strictTransportSecs = "max-age=31536000; includeSubDomains"
)

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setSecurityHeaders(w, r)
		next.ServeHTTP(w, r)
	})
}

func setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy", cspPolicy)
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", referrerPolicy)
	w.Header().Set("Permissions-Policy", permissionsPolicy)

	if requestIsHTTPS(r) {
		w.Header().Set("Strict-Transport-Security", strictTransportSecs)
	}
}

func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
