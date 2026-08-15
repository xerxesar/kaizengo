package appspec_test

import (
	"os"
	"path/filepath"
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestListNamesAt(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("appman/app.yaml", "name: appman\n")
	mustWrite("inventory/app.yaml", "name: inventory\n")
	mustWrite("notes/readme.txt", "not an app")
	if err := os.Mkdir(filepath.Join(root, "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	names, err := appspec.ListNamesAt(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "appman" || names[1] != "inventory" {
		t.Fatalf("ListNamesAt = %v, want [appman inventory]", names)
	}
}
