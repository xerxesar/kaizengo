package i18n

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed locale/*.po
var platformLocaleFS embed.FS

func init() {
	RegisterLocale(LocaleInfo{ID: "en", Name: "English", Dir: LTR})
	RegisterLocale(LocaleInfo{ID: "fa", Name: "فارسی", Dir: RTL})
	if err := LoadLocaleFS(platformLocaleFS, "locale"); err != nil {
		panic("i18n: platform locale: " + err.Error())
	}
}

// LoadLocaleFS loads every <locale>.po file under root in fsys.
func LoadLocaleFS(fsys fs.FS, root string) error {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return err
	}
	loaded := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".po") {
			continue
		}
		localeID := strings.TrimSuffix(e.Name(), ".po")
		data, err := fs.ReadFile(fsys, root+"/"+e.Name())
		if err != nil {
			return err
		}
		msgs, err := ParsePO(data)
		if err != nil {
			return fmt.Errorf("%s/%s: %w", root, e.Name(), err)
		}
		Register(localeID, msgs)
		loaded++
	}
	if loaded == 0 {
		return fmt.Errorf("i18n: no .po files in %s", root)
	}
	return nil
}
