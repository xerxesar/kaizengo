package dbmanager

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const (
	managerCookie     = "kg_dbmanager"
	managerSessionTTL = 12 * time.Hour
)

type managerSession struct {
	expires time.Time
}

type sessionStore struct {
	mu   sync.Mutex
	byID map[string]managerSession
}

func newSessionStore() *sessionStore {
	return &sessionStore{byID: map[string]managerSession{}}
}

func (s *sessionStore) create() (id string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	id = hex.EncodeToString(b[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[id] = managerSession{expires: time.Now().Add(managerSessionTTL)}
	return id, nil
}

func (s *sessionStore) valid(id string) bool {
	if id == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.byID[id]
	if !ok {
		return false
	}
	if time.Now().After(sess.expires) {
		delete(s.byID, id)
		return false
	}
	// sliding expiry
	sess.expires = time.Now().Add(managerSessionTTL)
	s.byID[id] = sess
	return true
}

func (s *sessionStore) revoke(id string) {
	if id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byID, id)
}

func setManagerCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{
		Name:     managerCookie,
		Value:    id,
		Path:     "/web/database",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(managerSessionTTL.Seconds()),
	})
}

func clearManagerCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     managerCookie,
		Value:    "",
		Path:     "/web/database",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func managerCookieID(r *http.Request) string {
	c, err := r.Cookie(managerCookie)
	if err != nil {
		return ""
	}
	return c.Value
}
