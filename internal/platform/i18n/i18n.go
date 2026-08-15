package i18n

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var (
	mu       sync.RWMutex
	locale   = "en"
	catalogs = map[string]map[string]string{}
)

// SetLocale selects the active locale (missing keys fall back to en).
func SetLocale(id string) {
	mu.Lock()
	defer mu.Unlock()
	locale = id
}

// Locale returns the active locale ID.
func Locale() string {
	mu.RLock()
	defer mu.RUnlock()
	return locale
}

// Register merges messages for a locale (apps and locale packs call this from init).
func Register(localeID string, messages map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	if catalogs[localeID] == nil {
		catalogs[localeID] = map[string]string{}
	}
	for k, v := range messages {
		catalogs[localeID][k] = v
	}
}

// Locales returns registered locale IDs sorted.
func Locales() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(catalogs))
	for id := range catalogs {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// T translates key for the active locale, then en, then returns the key.
func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()
	return lookup(key)
}

// Tf is T with fmt.Sprintf-style formatting.
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

// Bundle returns locale + messages for the given keys and/or key prefixes.
// Empty keys and prefixes returns an empty message map.
func Bundle(keys []string, prefixes ...string) (active string, messages map[string]string) {
	mu.RLock()
	defer mu.RUnlock()
	active = locale
	messages = map[string]string{}

	want := map[string]struct{}{}
	for _, k := range keys {
		if k != "" {
			want[k] = struct{}{}
		}
	}
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		for _, cat := range []map[string]string{catalogs[locale], catalogs["en"]} {
			if cat == nil {
				continue
			}
			for k := range cat {
				if strings.HasPrefix(k, prefix) {
					want[k] = struct{}{}
				}
			}
		}
	}

	for k := range want {
		messages[k] = lookup(k)
	}
	return active, messages
}

func lookup(key string) string {
	if m := catalogs[locale]; m != nil {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if m := catalogs["en"]; m != nil {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}
