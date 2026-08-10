package i18n

import (
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	locale   = "en"
	catalogs = map[string]map[string]string{
		"en": {
			"clock.title":      "Clock",
			"clock.subtitle":   "Local time, ticking live.",
			"clock.calendar":   "Calendar",
			"settings.title":   "Settings",
			"settings.locale":  "Locale",
			"settings.calendar": "Default calendar",
			"settings.shell":   "Shell title",
			"settings.save":    "Save",
			"settings.saved":   "Saved.",
		},
	}
)

// SetLocale selects the active locale (falls back to en for missing keys).
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

// Register merges messages for a locale (extension point for locale packs).
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
