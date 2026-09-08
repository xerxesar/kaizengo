package config

import (
	"os"
	"path/filepath"
	"strings"
)

func PostgresDSN() string {
	return os.Getenv("KaizenGo_POSTGRES_DSN")
}

func MongoURI() string {
	return os.Getenv("KaizenGo_MONGO_URI")
}

// DatabasesConfigPath returns the path to the file-backed DB catalog.
func DatabasesConfigPath() string {
	if p := strings.TrimSpace(os.Getenv("KaizenGo_DATABASES_CONFIG")); p != "" {
		return p
	}
	return filepath.Join(".kaizengo", "databases.json")
}
