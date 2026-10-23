package dbmanager

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHTTPSetupLoginAndGate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	swap := NewSwapHandler(BootstrapHandler())
	rt := NewRuntime(store, swap, func(ctx context.Context, dsn string) (*Platform, error) {
		return &Platform{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok:" + dsn))
		})}, nil
	})

	r := chi.NewRouter()
	rt.Mount(r)

	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/web/database/status", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status code %d: %s", res.Code, res.Body.String())
	}
	var st map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st["needs_setup"] != true {
		t.Fatalf("expected needs_setup: %#v", st)
	}

	body := bytes.NewBufferString(`{"password":"master-secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/web/database/setup", body)
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("setup code %d: %s", res.Code, res.Body.String())
	}
	cookie := res.Result().Cookies()
	if len(cookie) == 0 {
		t.Fatal("expected session cookie after setup")
	}

	// Without cookie, status is locked (no database list).
	res = httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/web/database/status", nil))
	_ = json.Unmarshal(res.Body.Bytes(), &st)
	if st["needs_setup"] != false || st["unlocked"] != false {
		t.Fatalf("expected locked status: %#v", st)
	}
	if _, ok := st["databases"]; ok {
		t.Fatalf("databases should be hidden when locked: %#v", st)
	}

	// List requires session.
	res = httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/web/database/list", nil))
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for list, got %d", res.Code)
	}

	// Login issues cookie.
	body = bytes.NewBufferString(`{"password":"master-secret"}`)
	req = httptest.NewRequest(http.MethodPost, "/web/database/login", body)
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("login %d: %s", res.Code, res.Body.String())
	}
	var loginCookie *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == managerCookie {
			loginCookie = c
		}
	}
	if loginCookie == nil {
		t.Fatal("expected manager cookie")
	}

	req = httptest.NewRequest(http.MethodGet, "/web/database/status", nil)
	req.AddCookie(loginCookie)
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	_ = json.Unmarshal(res.Body.Bytes(), &st)
	if st["unlocked"] != true {
		t.Fatalf("expected unlocked: %#v", st)
	}

	res = httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/web/database/manager", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("manager UI %d", res.Code)
	}
	if ct := res.Header().Get("Content-Type"); ct == "" || !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type: %s", ct)
	}
}

func TestHTTPImportMissingDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetupMasterPassword("master"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KaizenGo_POSTGRES_DSN", "postgres://nobody:x@127.0.0.1:1/ghost?sslmode=disable")

	swap := NewSwapHandler(BootstrapHandler())
	rt := NewRuntime(store, swap, func(ctx context.Context, dsn string) (*Platform, error) {
		return &Platform{Handler: http.NotFoundHandler()}, nil
	})
	r := chi.NewRouter()
	rt.Mount(r)

	// Unlock first.
	body := bytes.NewBufferString(`{"password":"master"}`)
	req := httptest.NewRequest(http.MethodPost, "/web/database/login", body)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	var c *http.Cookie
	for _, x := range res.Result().Cookies() {
		if x.Name == managerCookie {
			c = x
		}
	}
	if c == nil {
		t.Fatal("login cookie missing")
	}

	body = bytes.NewBufferString(`{"name":"ghost_db"}`)
	req = httptest.NewRequest(http.MethodPost, "/web/database/import", body)
	req.AddCookie(c)
	res = httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code == http.StatusOK {
		t.Fatalf("expected import failure, got %s", res.Body.String())
	}
}

func TestHTTPUseRebuild(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")
	t.Setenv("KaizenGo_POSTGRES_DSN", "postgres://u:p@localhost:6432/kaizengo?sslmode=disable")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetupMasterPassword("master"); err != nil {
		t.Fatal(err)
	}
	if !store.HasDatabase("demo") {
		if err := store.AddDatabase("demo", true); err != nil {
			t.Fatal(err)
		}
	}

	built := 0
	swap := NewSwapHandler(BootstrapHandler())
	rt := NewRuntime(store, swap, func(ctx context.Context, dsn string) (*Platform, error) {
		built++
		return &Platform{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("platform"))
		})}, nil
	})
	r := chi.NewRouter()
	rt.Mount(r)
	r.NotFound(swap.ServeHTTP)
	r.MethodNotAllowed(swap.ServeHTTP)

	body := bytes.NewBufferString(`{"name":"demo"}`)
	req := httptest.NewRequest(http.MethodPost, "/web/database/use", body)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("use %d: %s", res.Code, res.Body.String())
	}
	if built != 1 {
		t.Fatalf("expected 1 rebuild, got %d", built)
	}
	if rt.ActiveName() != "demo" {
		t.Fatalf("active=%q", rt.ActiveName())
	}
	if store.SelectedName() != "" {
		t.Fatalf("catalog should not persist selection, selected=%q", store.SelectedName())
	}
	var useCookie *http.Cookie
	for _, c := range res.Result().Cookies() {
		if c.Name == ClientDBCookie {
			useCookie = c
		}
	}
	if useCookie == nil || useCookie.Value != "demo" {
		t.Fatalf("expected kg_db cookie, got %#v", useCookie)
	}

	res = httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/anything", nil))
	if res.Body.String() != "platform" {
		t.Fatalf("swap handler body=%q", res.Body.String())
	}
}
