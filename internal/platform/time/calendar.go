package time

import (
	"fmt"
	"sort"
	"sync"
	gotime "time"
)

// Calendar is a platform time extension point (like a kernel clock driver).
type Calendar interface {
	ID() string
	Name() string
	Format(t gotime.Time) string
}

var (
	mu   sync.RWMutex
	cals = map[string]Calendar{}
)

// Register adds a calendar driver. Panics on duplicate IDs.
func Register(c Calendar) {
	if c == nil || c.ID() == "" {
		panic("platform/time: calendar requires a non-empty ID")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, ok := cals[c.ID()]; ok {
		panic(fmt.Sprintf("platform/time: duplicate calendar %q", c.ID()))
	}
	cals[c.ID()] = c
}

// Get returns a calendar by ID.
func Get(id string) (Calendar, bool) {
	mu.RLock()
	defer mu.RUnlock()
	c, ok := cals[id]
	return c, ok
}

// List returns registered calendars sorted by ID.
func List() []Calendar {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Calendar, 0, len(cals))
	for _, c := range cals {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}

// Format formats t with the given calendar ID (default gregorian).
func Format(calendarID string, t gotime.Time) (string, error) {
	if calendarID == "" {
		calendarID = "gregorian"
	}
	c, ok := Get(calendarID)
	if !ok {
		return "", fmt.Errorf("unknown calendar %q", calendarID)
	}
	return c.Format(t), nil
}
