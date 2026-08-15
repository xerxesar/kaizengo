package i18n

import "sort"

// TextDirection is the writing direction for a locale.
type TextDirection string

const (
	LTR TextDirection = "ltr"
	RTL TextDirection = "rtl"
)

// LocaleInfo describes a registered locale (id, display name, direction).
type LocaleInfo struct {
	ID   string
	Name string
	Dir  TextDirection
}

var localeMeta = map[string]LocaleInfo{}

// RegisterLocale records metadata for a locale (name + text direction).
// Call alongside Register / LoadPOFile. Unknown locales default to LTR.
func RegisterLocale(info LocaleInfo) {
	mu.Lock()
	defer mu.Unlock()
	if info.Dir == "" {
		info.Dir = LTR
	}
	if info.Name == "" {
		info.Name = info.ID
	}
	localeMeta[info.ID] = info
}

// Info returns metadata for a locale (falls back to id + LTR).
func Info(localeID string) LocaleInfo {
	mu.RLock()
	defer mu.RUnlock()
	if info, ok := localeMeta[localeID]; ok {
		return info
	}
	return LocaleInfo{ID: localeID, Name: localeID, Dir: LTR}
}

// ActiveInfo returns metadata for the active locale.
func ActiveInfo() LocaleInfo {
	return Info(Locale())
}

// Dir returns the text direction for the active locale.
func Dir() TextDirection {
	return ActiveInfo().Dir
}

// LocaleInfos returns all locales that have catalog messages, with metadata.
func LocaleInfos() []LocaleInfo {
	mu.RLock()
	defer mu.RUnlock()
	ids := make([]string, 0, len(catalogs))
	for id := range catalogs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]LocaleInfo, 0, len(ids))
	for _, id := range ids {
		if info, ok := localeMeta[id]; ok {
			out = append(out, info)
		} else {
			out = append(out, LocaleInfo{ID: id, Name: id, Dir: LTR})
		}
	}
	return out
}
