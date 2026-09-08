package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/codegen"
)

func runGenTypes(args []string) error {
	root, err := findModuleRoot()
	if err != nil {
		return err
	}
	appsRoot := filepath.Join(root, "apps")

	single := len(args) > 0 && !strings.HasPrefix(args[0], "-")
	var apps []string
	if single {
		apps = append(apps, args[0])
	} else {
		entries, err := os.ReadDir(appsRoot)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(appsRoot, e.Name(), "app.yaml")); err == nil {
				apps = append(apps, e.Name())
			}
		}
	}
	if len(apps) == 0 {
		return fmt.Errorf("no apps with app.yaml found")
	}

	var specs []appspec.AppSpec
	for _, name := range apps {
		appDir := filepath.Join(appsRoot, name)
		spec, err := appspec.LoadFile(filepath.Join(appDir, "app.yaml"))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		specs = append(specs, spec)
		if err := codegen.GenerateAppTypes(spec, appDir); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if len(spec.Models) == 0 {
			fmt.Printf("generated apps/%s/__types__ (no models)\n", name)
		} else {
			for _, m := range spec.Models {
				fmt.Printf("generated apps/%s/__types__/%s.go\n", name, m.Name)
			}
		}
		n, err := codegen.GenerateLocaleTemplate(spec, appDir)
		if err != nil {
			return fmt.Errorf("%s locale: %w", name, err)
		}
		if n > 0 {
			fmt.Printf("generated apps/%s/locale/template.pot (%d keys)\n", name, n)
		}
	}
	if !single {
		n, err := codegen.GeneratePlatformLocaleTemplate(root, specs)
		if err != nil {
			return fmt.Errorf("platform locale: %w", err)
		}
		fmt.Printf("generated internal/platform/i18n/locale/template.pot (%d keys)\n", n)
	}
	return nil
}
