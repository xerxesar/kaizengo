package dbmanager

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"kaizengo/internal/platform/config"
)

var dbNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// DatabaseEntry is one registered application database.
type DatabaseEntry struct {
	Name      string    `json:"name"`
	Imported  bool      `json:"imported,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Catalog is the on-disk database manager state (no server connection settings).
type Catalog struct {
	NewPassword        string          `json:"NEW_PASSWORD,omitempty"`
	MasterPasswordHash string          `json:"master_password_hash"`
	Selected           string          `json:"selected,omitempty"`
	Databases          []DatabaseEntry `json:"databases"`
}

// Store loads and persists the catalog file.
type Store struct {
	mu   sync.Mutex
	path string
	cat  Catalog
}

// OpenStore loads the catalog (or seeds selected DB name from KaizenGo_POSTGRES_DSN) and processes NEW_PASSWORD.
func OpenStore(path string) (*Store, error) {
	if path == "" {
		path = config.DatabasesConfigPath()
	}
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		s.cat = Catalog{Databases: []DatabaseEntry{}}
		if dsn := strings.TrimSpace(config.PostgresDSN()); dsn != "" {
			name, err := databaseNameFromDSN(dsn)
			if err != nil {
				return fmt.Errorf("seed catalog from KaizenGo_POSTGRES_DSN: %w", err)
			}
			s.cat.Selected = ""
			s.cat.Databases = []DatabaseEntry{{
				Name:      name,
				CreatedAt: time.Now().UTC(),
			}}
			if err := s.saveLocked(); err != nil {
				return err
			}
		}
		return s.processNewPasswordLocked()
	}

	if err := json.Unmarshal(b, &s.cat); err != nil {
		return fmt.Errorf("parse %s: %w", s.path, err)
	}
	if s.cat.Databases == nil {
		s.cat.Databases = []DatabaseEntry{}
	}
	// Selection is client-side only; do not keep a persisted server selection.
	s.cat.Selected = ""
	if err := s.processNewPasswordLocked(); err != nil {
		return err
	}
	// Rewrite so legacy "postgres" connection blobs / selected are dropped from disk.
	return s.saveLocked()
}

func (s *Store) processNewPasswordLocked() error {
	pw := strings.TrimSpace(s.cat.NewPassword)
	if pw == "" {
		return nil
	}
	hash, err := hashPassword(pw)
	if err != nil {
		return err
	}
	s.cat.MasterPasswordHash = hash
	s.cat.NewPassword = ""
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s.cat, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(s.path, b, 0o600)
}

// Snapshot returns a copy of the catalog.
func (s *Store) Snapshot() Catalog {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneCatalog(s.cat)
}

func cloneCatalog(c Catalog) Catalog {
	out := c
	out.Databases = append([]DatabaseEntry(nil), c.Databases...)
	return out
}

// NeedsSetup reports whether the master password has not been set.
func (s *Store) NeedsSetup() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cat.MasterPasswordHash == ""
}

// PostgresConfigured reports whether KaizenGo_POSTGRES_DSN is set and usable as a template.
func (s *Store) PostgresConfigured() bool {
	_, err := basePostgresURL()
	return err == nil
}

// ConnectionInfo returns non-secret fields from KaizenGo_POSTGRES_DSN for display.
func (s *Store) ConnectionInfo() (host string, port string, user string, sslmode string, ok bool) {
	u, err := basePostgresURL()
	if err != nil {
		return "", "", "", "", false
	}
	host = u.Hostname()
	port = u.Port()
	if u.User != nil {
		user = u.User.Username()
	}
	sslmode = u.Query().Get("sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}
	return host, port, user, sslmode, true
}

// SetupMasterPassword sets the master password when unset.
func (s *Store) SetupMasterPassword(password string) error {
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("master password is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cat.MasterPasswordHash != "" {
		return fmt.Errorf("master password already set")
	}
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	s.cat.MasterPasswordHash = hash
	return s.saveLocked()
}

// VerifyMasterPassword checks the master password.
func (s *Store) VerifyMasterPassword(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cat.MasterPasswordHash == "" {
		return fmt.Errorf("master password not set")
	}
	return comparePassword(s.cat.MasterPasswordHash, password)
}

// ChangeMasterPassword updates the master password.
func (s *Store) ChangeMasterPassword(oldPassword, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)
	if newPassword == "" {
		return fmt.Errorf("new password is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cat.MasterPasswordHash == "" {
		return fmt.Errorf("master password not set")
	}
	if err := comparePassword(s.cat.MasterPasswordHash, oldPassword); err != nil {
		return fmt.Errorf("invalid current password")
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	s.cat.MasterPasswordHash = hash
	return s.saveLocked()
}

// List returns registered databases and the selected name.
func (s *Store) List() (selected string, dbs []DatabaseEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cat.Selected, append([]DatabaseEntry(nil), s.cat.Databases...)
}

// SelectedName returns the currently selected database name.
func (s *Store) SelectedName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cat.Selected
}

// HasDatabase reports whether name is in the catalog.
func (s *Store) HasDatabase(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.cat.Databases {
		if d.Name == name {
			return true
		}
	}
	return false
}

// AddDatabase registers a database in the catalog.
func (s *Store) AddDatabase(name string, imported bool) error {
	if err := ValidateDBName(name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.cat.Databases {
		if d.Name == name {
			return fmt.Errorf("database %q already registered", name)
		}
	}
	s.cat.Databases = append(s.cat.Databases, DatabaseEntry{
		Name:      name,
		Imported:  imported,
		CreatedAt: time.Now().UTC(),
	})
	return s.saveLocked()
}

// RemoveDatabase removes a database from the catalog.
func (s *Store) RemoveDatabase(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.cat.Databases[:0]
	found := false
	for _, d := range s.cat.Databases {
		if d.Name == name {
			found = true
			continue
		}
		out = append(out, d)
	}
	if !found {
		return fmt.Errorf("database %q not registered", name)
	}
	s.cat.Databases = out
	return s.saveLocked()
}

// SetSelected records the selected database name.
func (s *Store) SetSelected(name string) error {
	if name != "" {
		if err := ValidateDBName(name); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if name != "" {
		found := false
		for _, d := range s.cat.Databases {
			if d.Name == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("database %q not registered", name)
		}
	}
	s.cat.Selected = name
	return s.saveLocked()
}

// ClearSelected clears the selected database.
func (s *Store) ClearSelected() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cat.Selected = ""
	return s.saveLocked()
}

// AdminDSN returns the platform DSN (KaizenGo_POSTGRES_DSN) for admin ops.
func (s *Store) AdminDSN() (string, error) {
	dsn := strings.TrimSpace(config.PostgresDSN())
	if dsn == "" {
		return "", fmt.Errorf("KaizenGo_POSTGRES_DSN is not set")
	}
	return dsn, nil
}

// DatabaseDSN returns a DSN for the named application database.
func (s *Store) DatabaseDSN(name string) (string, error) {
	if err := ValidateDBName(name); err != nil {
		return "", err
	}
	return dsnForDatabase(name)
}

// SelectedDSN returns the DSN for the selected database, or empty if none.
func (s *Store) SelectedDSN() (string, error) {
	s.mu.Lock()
	selected := s.cat.Selected
	s.mu.Unlock()
	if selected == "" {
		return "", nil
	}
	return dsnForDatabase(selected)
}

func basePostgresURL() (*url.URL, error) {
	dsn := strings.TrimSpace(config.PostgresDSN())
	if dsn == "" {
		return nil, fmt.Errorf("KaizenGo_POSTGRES_DSN is not set")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse KaizenGo_POSTGRES_DSN: %w", err)
	}
	if u.Scheme == "" {
		return nil, fmt.Errorf("KaizenGo_POSTGRES_DSN missing scheme")
	}
	return u, nil
}

func databaseNameFromDSN(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" || strings.Contains(name, "/") {
		return "", fmt.Errorf("DSN has no database name")
	}
	if err := ValidateDBName(name); err != nil {
		return "", err
	}
	return name, nil
}

func dsnForDatabase(dbName string) (string, error) {
	u, err := basePostgresURL()
	if err != nil {
		return "", err
	}
	cloned := *u
	cloned.Path = "/" + dbName
	cloned.RawPath = ""
	return cloned.String(), nil
}

// ValidateDBName enforces a safe PostgreSQL identifier subset.
func ValidateDBName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("database name is required")
	}
	if len(name) > 63 {
		return fmt.Errorf("database name too long (max 63)")
	}
	if name == "postgres" || name == "template0" || name == "template1" {
		return fmt.Errorf("database name %q is reserved", name)
	}
	if !dbNameRe.MatchString(name) {
		return fmt.Errorf("invalid database name %q (use lowercase letters, digits, underscore; start with a letter)", name)
	}
	return nil
}
