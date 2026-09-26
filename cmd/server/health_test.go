package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDB struct {
	err error
}

func (f fakeDB) PingContext(context.Context) error { return f.err }

func TestHealthOK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	HealthHandler(fakeDB{}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "ok\n" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestHealthDatabaseDown(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	HealthHandler(fakeDB{err: errors.New("refused")}).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestIndexShowsEnv(t *testing.T) {
	cfg := Config{Env: "dev", DBHost: "dev-postgres"}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	IndexHandler(cfg).ServeHTTP(rec, req)
	if rec.Body.String() != "env=dev\ndb_host=dev-postgres\n" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}
