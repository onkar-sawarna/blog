package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestInvalidPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/likes/", nil)
	rec := httptest.NewRecorder()
	Handler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestInvalidAction(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/likes/demo-post", strings.NewReader(`{"action":"nope"}`))
	rec := httptest.NewRecorder()
	Handler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestLocalLikeRoundTrip(t *testing.T) {
	t.Setenv("VERCEL", "")
	t.Setenv("UPSTASH_REDIS_REST_URL", "")
	t.Setenv("UPSTASH_REDIS_REST_TOKEN", "")
	t.Setenv("KV_REST_API_URL", "")
	t.Setenv("KV_REST_API_TOKEN", "")

	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	post := httptest.NewRequest(http.MethodPost, "/api/likes/demo-post", strings.NewReader(`{"action":"like"}`))
	rec := httptest.NewRecorder()
	Handler(rec, post)
	if rec.Code != http.StatusOK {
		t.Fatalf("like: %d %s", rec.Code, rec.Body.String())
	}
	var liked countBody
	if err := json.Unmarshal(rec.Body.Bytes(), &liked); err != nil {
		t.Fatal(err)
	}
	if liked.Count != 1 {
		t.Fatalf("count %d", liked.Count)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/likes/demo-post", nil)
	rec = httptest.NewRecorder()
	Handler(rec, get)
	if rec.Code != http.StatusOK {
		t.Fatalf("get: %d", rec.Code)
	}
	var got countBody
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Count != 1 {
		t.Fatalf("get count %d", got.Count)
	}
}
