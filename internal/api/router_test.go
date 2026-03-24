package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter_MediaPlaceholder(t *testing.T) {
	legacy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router := NewRouter(legacy)
	req := httptest.NewRequest(http.MethodGet, "/api/media", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("期望 501，得到 %d", w.Code)
	}
	if body := w.Body.String(); body == "" {
		t.Fatal("占位接口应该返回错误信息")
	}
}

func TestNewRouter_FallsBackToLegacyHandler(t *testing.T) {
	legacy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Fatalf("期望回退到 /login，得到 %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusTeapot)
	})

	router := NewRouter(legacy)
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Fatalf("期望 418，得到 %d", w.Code)
	}
}
