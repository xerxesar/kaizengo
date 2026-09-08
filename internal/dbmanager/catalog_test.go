package dbmanager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestValidateDBName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"kaizengo", false},
		{"my_db1", false},
		{"", true},
		{"Postgres", true},
		{"postgres", true},
		{"template0", true},
		{"1bad", true},
		{"has-dash", true},
		{"has space", true},
	}
	for _, tc := range cases {
		err := ValidateDBName(tc.name)
		if tc.wantErr && err == nil {
			t.Fatalf("%q: expected error", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%q: unexpected error: %v", tc.name, err)
		}
	}
}

func TestNewPasswordAutoHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")
	raw := Catalog{NewPassword: "secret-master"}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.NeedsSetup() {
		t.Fatal("expected master password to be set from NEW_PASSWORD")
	}
	if err := store.VerifyMasterPassword("secret-master"); err != nil {
		t.Fatalf("verify: %v", err)
	}

	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved Catalog
	if err := json.Unmarshal(disk, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.NewPassword != "" {
		t.Fatalf("NEW_PASSWORD should be cleared, got %q", saved.NewPassword)
	}
	if saved.MasterPasswordHash == "" {
		t.Fatal("expected hash on disk")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(saved.MasterPasswordHash), []byte("secret-master")); err != nil {
		t.Fatalf("disk hash mismatch: %v", err)
	}
	if strings.Contains(string(disk), `"postgres"`) {
		t.Fatal("catalog must not persist postgres connection info")
	}
}

func TestCatalogRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")
	t.Setenv("KaizenGo_POSTGRES_DSN", "postgres://admin:x@db.example:5432/unused?sslmode=require")
	if err := os.WriteFile(path, []byte(`{"master_password_hash":"","selected":"","databases":[]}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetupMasterPassword("pw"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddDatabase("alpha", false); err != nil {
		t.Fatal(err)
	}
	if err := store.AddDatabase("beta", true); err != nil {
		t.Fatal(err)
	}
	if err := store.SetSelected("alpha"); err != nil {
		t.Fatal(err)
	}

	store2, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	sel, dbs := store2.List()
	if sel != "" || len(dbs) != 2 {
		t.Fatalf("got selected=%q dbs=%v (selection must not persist)", sel, dbs)
	}
	dsn, err := store2.DatabaseDSN("alpha")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"db.example", "5432", "alpha", "require"} {
		if !strings.Contains(dsn, part) {
			t.Fatalf("unexpected dsn %q (missing %q)", dsn, part)
		}
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(disk), `"postgres"`) || strings.Contains(string(disk), "db.example") {
		t.Fatalf("connection info leaked into catalog: %s", disk)
	}
	if strings.Contains(string(disk), `"selected": "alpha"`) {
		t.Fatalf("selected should be cleared on load: %s", disk)
	}
}

func TestSeedFromDSNEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")
	t.Setenv("KaizenGo_POSTGRES_DSN", "postgres://u:p@127.0.0.1:6432/acme?sslmode=disable")

	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	sel, dbs := store.List()
	if sel != "" || len(dbs) != 1 || dbs[0].Name != "acme" {
		t.Fatalf("seed failed: selected=%q dbs=%v", sel, dbs)
	}
	host, port, user, _, ok := store.ConnectionInfo()
	if !ok || host != "127.0.0.1" || port != "6432" || user != "u" {
		t.Fatalf("connection info: %s %s %s ok=%v", host, port, user, ok)
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(disk), `"postgres"`) {
		t.Fatalf("seed wrote connection into catalog: %s", disk)
	}
}

func TestLegacyPostgresFieldStripped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")
	legacy := `{
  "master_password_hash": "",
  "postgres": {"host":"localhost","port":6432,"user":"kaizengo","password":"secret","sslmode":"disable"},
  "selected": "kaizengo",
  "databases": [{"name":"kaizengo","created_at":"2026-01-01T00:00:00Z"}]
}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if store.SelectedName() != "" {
		t.Fatalf("selected=%q (should be cleared)", store.SelectedName())
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(disk), "secret") || strings.Contains(string(disk), `"postgres"`) {
		t.Fatalf("legacy postgres not stripped: %s", disk)
	}
}
