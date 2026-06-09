package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	srv := &server{maxRequestBody: 64}

	payload := `{"word":"` + strings.Repeat("a", 128) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/games/x/move", strings.NewReader(payload))
	rec := httptest.NewRecorder()

	var reqBody moveRequest
	err := srv.decodeJSON(rec, req, &reqBody)
	if err == nil {
		t.Fatal("expected oversized body to fail")
	}
	if !errors.Is(err, errRequestBodyTooLarge) {
		t.Fatalf("got err %v, want errRequestBodyTooLarge", err)
	}
}

func TestDecodeJSONAcceptsValidBody(t *testing.T) {
	srv := &server{maxRequestBody: 1024}

	req := httptest.NewRequest(http.MethodPost, "/api/games", strings.NewReader(`{"start":"cat","end":"dog","difficulty":"easy"}`))
	rec := httptest.NewRecorder()

	var reqBody createGameRequest
	if err := srv.decodeJSON(rec, req, &reqBody); err != nil {
		t.Fatalf("decodeJSON: %v", err)
	}
	if reqBody.Start != "cat" || reqBody.End != "dog" {
		t.Fatalf("unexpected payload: %+v", reqBody)
	}
}

func TestWriteDecodeErrorUses413ForLargeBody(t *testing.T) {
	rec := httptest.NewRecorder()
	writeDecodeError(rec, errRequestBodyTooLarge)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", rec.Code)
	}
}
