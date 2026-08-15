package appspec_test

import (
	"os"
	"path/filepath"
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestLoadBundledApps(t *testing.T) {
	root := findRoot(t)
	t.Chdir(root)
	spec, err := appspec.LoadApp("inventory")
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Models) != 19 {
		t.Fatalf("inventory models=%d want 19", len(spec.Models))
	}
	if len(spec.Views) != 18 {
		t.Fatalf("inventory pages=%d want 18", len(spec.Views))
	}
	for _, name := range []string{"identity", "hellospec", "settings", "appman", "status", "typesense", "core"} {
		if _, err := appspec.LoadApp(name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func findRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "apps", "inventory", "app.yaml")); err == nil {
				return dir
			}
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("repo root not found")
	return ""
}
