package config

import "sync"

// Process-wide platform/core settings (mutable at runtime via the settings app).
var (
	mu sync.RWMutex

	defaultCalendar = "gregorian"
	shellTitle      = "KaizenGo"
)

func DefaultCalendar() string {
	mu.RLock()
	defer mu.RUnlock()
	return defaultCalendar
}

func SetDefaultCalendar(id string) {
	mu.Lock()
	defer mu.Unlock()
	if id != "" {
		defaultCalendar = id
	}
}

func ShellTitle() string {
	mu.RLock()
	defer mu.RUnlock()
	return shellTitle
}

func SetShellTitle(title string) {
	mu.Lock()
	defer mu.Unlock()
	if title != "" {
		shellTitle = title
	}
}
