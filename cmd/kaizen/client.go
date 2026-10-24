package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type client struct {
	base string
	db   string
	u    *url.URL
	jar  *cookiejar.Jar
	http *http.Client
}

type shellResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output"`
	Data   string `json:"data"`
}

// sessionFile maps server base URL → cookies (session + selected database).
type sessionFile map[string]storedSession

type storedSession struct {
	DB      string         `json:"db,omitempty"`
	Cookies []storedCookie `json:"cookies"`
}

type storedCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func newClient(base, db string) (*client, error) {
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid --url %q", base)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	c := &client{base: base, db: db, u: u, jar: jar, http: &http.Client{Jar: jar, Timeout: 60 * time.Second}}
	sessions, _ := loadSessions()
	if s, ok := sessions[base]; ok {
		cookies := make([]*http.Cookie, 0, len(s.Cookies))
		for _, sc := range s.Cookies {
			cookies = append(cookies, &http.Cookie{Name: sc.Name, Value: sc.Value, Path: "/"})
		}
		jar.SetCookies(u, cookies)
		if c.db == "" {
			c.db = s.DB
		}
	}
	return c, nil
}

func (c *client) login(email, password string) (string, error) {
	if c.db != "" {
		body, _ := json.Marshal(map[string]string{"name": c.db})
		if _, err := c.post("/web/database/use", body); err != nil {
			return "", fmt.Errorf("select database %q: %w", c.db, err)
		}
	}
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	raw, err := c.post("/auth/login", body)
	if err != nil {
		return "", err
	}
	var user struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	_ = json.Unmarshal(raw, &user)
	if err := c.save(); err != nil {
		return "", err
	}
	if user.Name != "" {
		return user.Name, nil
	}
	return email, nil
}

func (c *client) logout() error {
	_, err := c.post("/auth/logout", nil)
	sessions, _ := loadSessions()
	delete(sessions, c.base)
	if serr := writeSessions(sessions); serr != nil {
		return serr
	}
	return err
}

func (c *client) shell(line string) (*shellResult, error) {
	body, _ := json.Marshal(map[string]any{
		"query":     "mutation($line: String!) { platformShell(line: $line) { ok output data } }",
		"variables": map[string]any{"line": line},
	})
	raw, err := c.post("/graphql", body)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Data struct {
			PlatformShell *shellResult `json:"platformShell"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unexpected response: %w", err)
	}
	if len(resp.Errors) > 0 {
		msgs := make([]string, len(resp.Errors))
		for i, e := range resp.Errors {
			msgs[i] = e.Message
		}
		return nil, errors.New(strings.Join(msgs, "; "))
	}
	if resp.Data.PlatformShell == nil {
		return nil, errors.New("empty response (run: kaizen login)")
	}
	return resp.Data.PlatformShell, nil
}

func (c *client) post(path string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = res.Status
		}
		if res.StatusCode == http.StatusUnauthorized && path != "/auth/login" {
			msg += " (run: kaizen login)"
		}
		return nil, fmt.Errorf("%s: %s", path, msg)
	}
	return raw, nil
}

func (c *client) save() error {
	sessions, _ := loadSessions()
	if sessions == nil {
		sessions = sessionFile{}
	}
	s := storedSession{DB: c.db}
	for _, ck := range c.jar.Cookies(c.u) {
		s.Cookies = append(s.Cookies, storedCookie{Name: ck.Name, Value: ck.Value})
	}
	sessions[c.base] = s
	return writeSessions(sessions)
}

func sessionPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "kaizen", "session.json"), nil
}

func loadSessions() (sessionFile, error) {
	path, err := sessionPath()
	if err != nil {
		return sessionFile{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return sessionFile{}, err
	}
	out := sessionFile{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return sessionFile{}, err
	}
	return out, nil
}

func writeSessions(s sessionFile) error {
	path, err := sessionPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}
