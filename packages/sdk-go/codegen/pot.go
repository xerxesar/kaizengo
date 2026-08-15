package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"kaizengo/packages/sdk-go/appspec"
)

var (
	callKeyRe = regexp.MustCompile(`(?:\b(?:t|i18n\.t|i18n\.Tf?|i18n\.Errorf?|labelField)\(|\.(?:T|Tf)\()\s*['"` + "`" + `]([a-z][a-zA-Z0-9_]*\.[a-zA-Z0-9_.]*)['"` + "`" + `]`)
	attrKeyRe = regexp.MustCompile(`(?:emptyKey|labelKey|TitleKey|titleKey)\s*[:=]\s*['"]([a-z][a-zA-Z0-9_]*\.[a-zA-Z0-9_.]*)['"]`)
	yamlKeyRe = regexp.MustCompile(`(?:labelKey|emptyKey):\s*([a-z][a-zA-Z0-9_]*\.[a-zA-Z0-9_.]*)`)
	propKeyRe = regexp.MustCompile(`\bkey:\s*['"]([a-z][a-zA-Z0-9_]*\.[a-zA-Z0-9_.]*)['"]`)
)

var skipDirNames = map[string]struct{}{
	"node_modules": {},
	"dist":         {},
	"__types__":    {},
	".git":         {},
}

// GenerateLocaleTemplate writes apps/<app>/locale/template.pot from the spec and source.
func GenerateLocaleTemplate(spec appspec.AppSpec, appDir string) (int, error) {
	if !spec.EnableI18n {
		return 0, nil
	}
	keys := collectSpecKeys(spec)
	if err := collectSourceKeys(appDir, keys); err != nil {
		return 0, err
	}
	outDir := filepath.Join(appDir, "locale")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, err
	}
	body := renderPOT(spec.Name, keys)
	if err := os.WriteFile(filepath.Join(outDir, "template.pot"), []byte(body), 0o644); err != nil {
		return 0, err
	}
	return len(keys), nil
}

// GeneratePlatformLocaleTemplate writes internal/platform/i18n/locale/template.pot (nav keys).
func GeneratePlatformLocaleTemplate(root string, specs []appspec.AppSpec) (int, error) {
	keys := map[string][]string{}
	for _, spec := range specs {
		label := strings.TrimSpace(spec.Nav.LabelKey)
		if label == "" {
			label = "nav." + spec.Name
		}
		addKey(keys, label, "app.yaml")
	}
	platformDir := filepath.Join(root, "internal", "platform")
	if err := collectSourceKeys(platformDir, keys); err != nil {
		return 0, err
	}
	outDir := filepath.Join(root, "internal", "platform", "i18n", "locale")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return 0, err
	}
	body := renderPOT("platform", keys)
	if err := os.WriteFile(filepath.Join(outDir, "template.pot"), []byte(body), 0o644); err != nil {
		return 0, err
	}
	return len(keys), nil
}

func collectSpecKeys(spec appspec.AppSpec) map[string][]string {
	keys := map[string][]string{}
	app := spec.Name
	src := "app.yaml"

	nav := strings.TrimSpace(spec.Nav.LabelKey)
	if nav == "" {
		nav = "nav." + app
	}
	addKey(keys, nav, src)
	addKey(keys, app+".title", src)
	addKey(keys, app+".subtitle", src)
	addKey(keys, app+".ping", src)

	if len(spec.Models) > 0 {
		addKey(keys, app+".create", src)
		addKey(keys, app+".save", src)
		addKey(keys, app+".created", src)
		addKey(keys, app+".saved", src)
		addKey(keys, app+".empty", src)
		addKey(keys, app+".new_placeholder", src)
	}

	addMenuKeys(keys, spec.Menus, src)
	for _, v := range spec.Exports.Views {
		addKey(keys, strings.TrimSpace(v.LabelKey), src)
	}
	for _, m := range spec.Exports.Menus {
		addKey(keys, strings.TrimSpace(m.LabelKey), src)
	}

	seenFields := map[string]struct{}{}
	for _, model := range spec.Models {
		if model.Search != nil {
			addKey(keys, app+"."+model.Name+".search", src)
		}
		if model.Internal {
			addKey(keys, app+".error."+model.Name+".internal", src)
		}
		for _, f := range model.Fields {
			if _, ok := seenFields[f.Name]; !ok {
				addKey(keys, app+".field."+f.Name, src)
				seenFields[f.Name] = struct{}{}
			}
			if f.IsEnum() {
				for _, v := range f.Values {
					addKey(keys, app+".enum."+model.Name+"."+f.Name+"."+v, src)
				}
			}
		}
	}
	return keys
}

func addMenuKeys(keys map[string][]string, menus []appspec.MenuSpec, src string) {
	for _, m := range menus {
		addKey(keys, strings.TrimSpace(m.LabelKey), src)
		addMenuKeys(keys, m.Children, src)
	}
}

func collectSourceKeys(root string, keys map[string][]string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, skip := skipDirNames[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".svelte", ".ts", ".js", ".go", ".yaml", ".yml":
		default:
			return nil
		}
		if d.Name() == "template.pot" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > 1<<20 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		extractKeysFromSource(string(data), rel, keys)
		return nil
	})
}

func extractKeysFromSource(src, rel string, keys map[string][]string) {
	for _, re := range []*regexp.Regexp{callKeyRe, attrKeyRe, yamlKeyRe, propKeyRe} {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			if len(m) > 1 {
				addKey(keys, m[1], rel)
			}
		}
	}
}

func addKey(keys map[string][]string, key, src string) {
	key = strings.TrimSpace(key)
	if key == "" || !strings.Contains(key, ".") {
		return
	}
	for _, existing := range keys[key] {
		if existing == src {
			return
		}
	}
	keys[key] = append(keys[key], src)
}

func renderPOT(project string, keys map[string][]string) string {
	names := make([]string, 0, len(keys))
	for k := range keys {
		names = append(names, k)
	}
	sort.Strings(names)

	var b strings.Builder
	b.WriteString("# Generated by kaizengo gen-types. DO NOT EDIT.\n")
	b.WriteString("msgid \"\"\n")
	b.WriteString("msgstr \"\"\n")
	b.WriteString(fmt.Sprintf("%s\n", strconv.Quote("Project-Id-Version: "+project+"\n")))
	b.WriteString(strconv.Quote("MIME-Version: 1.0\n") + "\n")
	b.WriteString(strconv.Quote("Content-Type: text/plain; charset=UTF-8\n") + "\n")
	b.WriteString(strconv.Quote("Content-Transfer-Encoding: 8bit\n") + "\n")
	b.WriteByte('\n')

	for _, key := range names {
		refs := append([]string(nil), keys[key]...)
		sort.Strings(refs)
		b.WriteString("#: ")
		b.WriteString(strings.Join(refs, " "))
		b.WriteByte('\n')
		b.WriteString("msgid ")
		b.WriteString(strconv.Quote(key))
		b.WriteByte('\n')
		b.WriteString("msgstr \"\"\n\n")
	}
	return b.String()
}
