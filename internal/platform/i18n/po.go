package i18n

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ParsePO parses gettext .po bytes into msgid → msgstr.
// msgid is used as the catalog key (msgctxt is ignored for lookup).
// Fuzzy entries and empty msgstr are skipped.
func ParsePO(data []byte) (map[string]string, error) {
	out := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	// Allow long lines in PO files.
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	var (
		msgid  strings.Builder
		msgstr strings.Builder
		state  string // "", "msgid", "msgstr"
		fuzzy  bool
	)

	flush := func() {
		id := msgid.String()
		str := msgstr.String()
		if id != "" && str != "" && !fuzzy {
			out[id] = str
		}
		msgid.Reset()
		msgstr.Reset()
		state = ""
		fuzzy = false
	}

	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)

		if trim == "" {
			flush()
			continue
		}
		if strings.HasPrefix(trim, "#,") && strings.Contains(trim, "fuzzy") {
			fuzzy = true
			continue
		}
		if strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "msgctxt ") {
			// Context ignored; keys are msgid.
			state = ""
			continue
		}
		if strings.HasPrefix(trim, "msgid ") {
			if state == "msgstr" {
				flush()
			}
			state = "msgid"
			msgid.Reset()
			s, err := unquotePO(strings.TrimPrefix(trim, "msgid "))
			if err != nil {
				return nil, err
			}
			msgid.WriteString(s)
			continue
		}
		if strings.HasPrefix(trim, "msgstr ") {
			state = "msgstr"
			msgstr.Reset()
			s, err := unquotePO(strings.TrimPrefix(trim, "msgstr "))
			if err != nil {
				return nil, err
			}
			msgstr.WriteString(s)
			continue
		}
		if strings.HasPrefix(trim, "\"") {
			s, err := unquotePO(trim)
			if err != nil {
				return nil, err
			}
			switch state {
			case "msgid":
				msgid.WriteString(s)
			case "msgstr":
				msgstr.WriteString(s)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	flush()
	// Header entry uses empty msgid — drop it.
	delete(out, "")
	return out, nil
}

func unquotePO(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	v, err := strconv.Unquote(s)
	if err != nil {
		return "", fmt.Errorf("po string: %w", err)
	}
	return v, nil
}

// LoadPOFile parses a .po file and merges messages into localeID.
func LoadPOFile(localeID, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	msgs, err := ParsePO(data)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	Register(localeID, msgs)
	return nil
}

// LoadLocaleDir loads every <locale>.po file in dir (e.g. en.po, fa.po).
func LoadLocaleDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	loaded := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".po") {
			continue
		}
		localeID := strings.TrimSuffix(e.Name(), ".po")
		if err := LoadPOFile(localeID, filepath.Join(dir, e.Name())); err != nil {
			return err
		}
		loaded++
	}
	if loaded == 0 {
		return fmt.Errorf("i18n: no .po files in %s", dir)
	}
	return nil
}

// MustLoadLocaleDir is LoadLocaleDir that panics on error (for init/Setup).
func MustLoadLocaleDir(dir string) {
	if err := LoadLocaleDir(dir); err != nil {
		panic(err)
	}
}
