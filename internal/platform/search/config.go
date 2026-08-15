package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ModelIndexConfig is runtime search indexing settings for one model.
type ModelIndexConfig struct {
	App        string   `json:"app"`
	Model      string   `json:"model"`
	Collection string   `json:"collection"`
	Enabled    bool     `json:"enabled"`
	Fields     []string `json:"fields"`
}

var (
	cfgMu    sync.RWMutex
	cfgPath  = defaultConfigPath()
	overrides = map[string]ModelIndexConfig{} // key: app.model
)

func defaultConfigPath() string {
	if p := strings.TrimSpace(os.Getenv("KaizenGo_SEARCH_CONFIG")); p != "" {
		return p
	}
	return filepath.Join(".kaizengo", "search.json")
}

func configKey(app, model string) string {
	return app + "." + model
}

// LoadConfig reads persisted search index overrides from disk.
func LoadConfig() error {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			overrides = map[string]ModelIndexConfig{}
			return nil
		}
		return err
	}
	var list []ModelIndexConfig
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	overrides = map[string]ModelIndexConfig{}
	for _, c := range list {
		if c.App == "" || c.Model == "" {
			continue
		}
		overrides[configKey(c.App, c.Model)] = c
	}
	return nil
}

func saveConfigLocked() error {
	list := make([]ModelIndexConfig, 0, len(overrides))
	for _, c := range overrides {
		list = append(list, c)
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, b, 0o644)
}

// GetOverride returns a persisted override if present.
func GetOverride(app, model string) (ModelIndexConfig, bool) {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	c, ok := overrides[configKey(app, model)]
	return c, ok
}

// SetOverride saves runtime indexing settings for a model.
func SetOverride(cfg ModelIndexConfig) error {
	if cfg.App == "" || cfg.Model == "" {
		return nil
	}
	cfgMu.Lock()
	defer cfgMu.Unlock()
	overrides[configKey(cfg.App, cfg.Model)] = cfg
	return saveConfigLocked()
}
