package dbmanager

import (
	"path/filepath"
	"testing"
)

func TestMasterPasswordSetupOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "databases.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !store.NeedsSetup() {
		t.Fatal("expected needs setup")
	}
	if err := store.SetupMasterPassword("one"); err != nil {
		t.Fatal(err)
	}
	if store.NeedsSetup() {
		t.Fatal("expected setup complete")
	}
	if err := store.SetupMasterPassword("two"); err == nil {
		t.Fatal("expected second setup to fail")
	}
	if err := store.VerifyMasterPassword("one"); err != nil {
		t.Fatal(err)
	}
	if err := store.VerifyMasterPassword("wrong"); err == nil {
		t.Fatal("expected bad password to fail")
	}
	if err := store.ChangeMasterPassword("one", "two"); err != nil {
		t.Fatal(err)
	}
	if err := store.VerifyMasterPassword("two"); err != nil {
		t.Fatal(err)
	}
}
