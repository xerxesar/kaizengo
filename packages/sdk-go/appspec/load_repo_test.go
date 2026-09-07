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
	for _, name := range []string{"identity", "hellospec", "settings", "appman", "status", "typesense", "core", "permissions", "auth"} {
		if _, err := appspec.LoadApp(name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	spec, err := appspec.LoadApp("hellospec")
	if err != nil {
		t.Fatal(err)
	}
	if len(spec.Models) < 1 {
		t.Fatalf("hellospec models=%d want >= 1", len(spec.Models))
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
			if _, err := os.Stat(filepath.Join(dir, "apps", "hellospec", "app.yaml")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("repo root not found")
	return ""
}
