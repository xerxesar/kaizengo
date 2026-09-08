package dbmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type apiError struct {
	status int
	msg    string
}

func (e *apiError) Error() string { return e.msg }

// Mount registers /web/database routes on r.
func (rt *Runtime) Mount(r chi.Router) {
	r.Route("/web/database", func(r chi.Router) {
		r.Get("/manager", serveManagerUI)
		r.Get("/status", rt.handleStatus)
		r.Post("/setup", rt.handleSetup)
		r.Post("/login", rt.handleLogin)
		r.Post("/logout", rt.handleLogout)
		// Client-driven activation (no master password; mirrors Odoo db picker).
		r.Post("/use", rt.handleUse)

		r.Group(func(r chi.Router) {
			r.Use(rt.requireSession)
			r.Get("/list", rt.handleList)
			r.Post("/create", rt.handleCreate)
			r.Post("/import", rt.handleImport)
			r.Post("/duplicate", rt.handleDuplicate)
			r.Post("/drop", rt.handleDrop)
			r.Get("/backup", rt.handleBackup)
			r.Post("/restore", rt.handleRestore)
			r.Post("/change_password", rt.handleChangePassword)
		})
	})
}

func (rt *Runtime) unlocked(r *http.Request) bool {
	return rt.sessions != nil && rt.sessions.valid(managerCookieID(r))
}

func (rt *Runtime) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rt.unlocked(r) {
			writeErr(w, &apiError{status: http.StatusUnauthorized, msg: "master password required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rt *Runtime) issueSession(w http.ResponseWriter) error {
	id, err := rt.sessions.create()
	if err != nil {
		return err
	}
	setManagerCookie(w, id)
	return nil
}

func (rt *Runtime) handleStatus(w http.ResponseWriter, r *http.Request) {
	needsSetup := rt.Store.NeedsSetup()
	unlocked := !needsSetup && rt.unlocked(r)
	payload := map[string]any{
		"needs_setup": needsSetup,
		"unlocked":    unlocked,
		"active":      rt.ActiveName(),
		"client_db":   ClientDBFromRequest(r),
	}
	if !unlocked {
		writeJSON(w, http.StatusOK, payload)
		return
	}

	_, dbs := rt.Store.List()
	host, port, user, sslmode, ok := rt.Store.ConnectionInfo()
	payload["databases"] = dbs
	payload["postgres_configured"] = rt.Store.PostgresConfigured()
	if ok {
		payload["postgres"] = map[string]any{
			"host":    host,
			"port":    port,
			"user":    user,
			"sslmode": sslmode,
			"source":  "KaizenGo_POSTGRES_DSN",
		}
	}
	writeJSON(w, http.StatusOK, payload)
}

func (rt *Runtime) handleList(w http.ResponseWriter, r *http.Request) {
	_, dbs := rt.Store.List()
	writeJSON(w, http.StatusOK, map[string]any{
		"active":    rt.ActiveName(),
		"client_db": ClientDBFromRequest(r),
		"databases": dbs,
	})
}

func (rt *Runtime) handleUse(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	if err := rt.UseDatabase(ctx, name); err != nil {
		writeErr(w, err)
		return
	}
	SetClientDBCookie(w, name)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active": name})
}

func (rt *Runtime) handleSetup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if err := rt.Store.SetupMasterPassword(body.Password); err != nil {
		writeErr(w, err)
		return
	}
	if err := rt.issueSession(w); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "unlocked": true})
}

func (rt *Runtime) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if rt.Store.NeedsSetup() {
		writeErr(w, &apiError{status: http.StatusForbidden, msg: "master password not set; complete setup first"})
		return
	}
	if err := rt.Store.VerifyMasterPassword(body.Password); err != nil {
		writeErr(w, &apiError{status: http.StatusUnauthorized, msg: err.Error()})
		return
	}
	if err := rt.issueSession(w); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "unlocked": true})
}

func (rt *Runtime) handleLogout(w http.ResponseWriter, r *http.Request) {
	rt.sessions.revoke(managerCookieID(r))
	clearManagerCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "unlocked": false})
}

func (rt *Runtime) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Select bool   `json:"select"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	if err := ValidateDBName(name); err != nil {
		writeErr(w, err)
		return
	}
	ctx := r.Context()
	if err := rt.Admin().CreateDatabase(ctx, name); err != nil {
		writeErr(w, err)
		return
	}
	if err := rt.Store.AddDatabase(name, false); err != nil {
		writeErr(w, err)
		return
	}
	if body.Select {
		if err := rt.UseDatabase(ctx, name); err != nil {
			writeErr(w, fmt.Errorf("created but activate failed: %w", err))
			return
		}
		SetClientDBCookie(w, name)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name})
}

func (rt *Runtime) handleImport(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Select bool   `json:"select"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	if err := ValidateDBName(name); err != nil {
		writeErr(w, err)
		return
	}
	ctx := r.Context()
	exists, err := rt.Admin().DatabaseExists(ctx, name)
	if err != nil {
		writeErr(w, err)
		return
	}
	if !exists {
		writeErr(w, &apiError{status: http.StatusBadRequest, msg: fmt.Sprintf("database %q does not exist on the server (import it after it exists on Postgres)", name)})
		return
	}
	if err := rt.Store.AddDatabase(name, true); err != nil {
		writeErr(w, err)
		return
	}
	if body.Select {
		if err := rt.UseDatabase(ctx, name); err != nil {
			writeErr(w, fmt.Errorf("imported but activate failed: %w", err))
			return
		}
		SetClientDBCookie(w, name)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name})
}

func (rt *Runtime) handleDuplicate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source string `json:"source"`
		Name   string `json:"name"`
		Select bool   `json:"select"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	source := strings.TrimSpace(body.Source)
	name := strings.TrimSpace(body.Name)
	if !rt.Store.HasDatabase(source) {
		writeErr(w, &apiError{status: http.StatusBadRequest, msg: fmt.Sprintf("source database %q not registered", source)})
		return
	}

	ctx := r.Context()
	wasActive := rt.ActiveName() == source
	if wasActive {
		rt.EnterBootstrap()
	}

	if err := rt.Admin().DuplicateDatabase(ctx, source, name); err != nil {
		if wasActive {
			_ = rt.UseDatabase(ctx, source)
		}
		writeErr(w, err)
		return
	}
	if err := rt.Store.AddDatabase(name, false); err != nil {
		writeErr(w, err)
		return
	}
	if body.Select {
		if err := rt.UseDatabase(ctx, name); err != nil {
			writeErr(w, fmt.Errorf("duplicated but activate failed: %w", err))
			return
		}
		SetClientDBCookie(w, name)
	} else if wasActive {
		if err := rt.UseDatabase(ctx, source); err != nil {
			writeErr(w, fmt.Errorf("duplicated but failed to reactivate source: %w", err))
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name})
}

func (rt *Runtime) handleDrop(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	name := strings.TrimSpace(body.Name)
	if rt.ActiveName() == name {
		writeErr(w, &apiError{status: http.StatusBadRequest, msg: "cannot drop the active database; switch to another first"})
		return
	}
	if !rt.Store.HasDatabase(name) {
		writeErr(w, &apiError{status: http.StatusBadRequest, msg: fmt.Sprintf("database %q not registered", name)})
		return
	}
	ctx := r.Context()
	if err := rt.Store.RemoveDatabase(name); err != nil {
		writeErr(w, err)
		return
	}
	if err := rt.Admin().DropDatabase(ctx, name); err != nil {
		_ = rt.Store.AddDatabase(name, false)
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (rt *Runtime) handleBackup(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if !rt.Store.HasDatabase(name) {
		writeErr(w, &apiError{status: http.StatusBadRequest, msg: fmt.Sprintf("database %q not registered", name)})
		return
	}
	dsn, err := rt.Store.DatabaseDSN(name)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/sql; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.sql"`, name))
	if err := DumpDatabase(dsn, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (rt *Runtime) handleRestore(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(512 << 20); err != nil {
		writeErr(w, &apiError{status: http.StatusBadRequest, msg: "invalid multipart form: " + err.Error()})
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if err := ValidateDBName(name); err != nil {
		writeErr(w, err)
		return
	}
	selectAfter := r.FormValue("select") == "true" || r.FormValue("select") == "1"

	file, _, err := r.FormFile("dump")
	if err != nil {
		writeErr(w, &apiError{status: http.StatusBadRequest, msg: "dump file is required"})
		return
	}
	defer file.Close()

	path, cleanup, err := WriteTempDump(file)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer cleanup()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	if err := rt.Admin().CreateDatabase(ctx, name); err != nil {
		writeErr(w, err)
		return
	}
	dsn, err := rt.Store.DatabaseDSN(name)
	if err != nil {
		_ = rt.Admin().DropDatabase(ctx, name)
		writeErr(w, err)
		return
	}
	if err := RestoreDatabase(dsn, path); err != nil {
		_ = rt.Admin().DropDatabase(ctx, name)
		writeErr(w, err)
		return
	}
	if err := rt.Store.AddDatabase(name, true); err != nil {
		writeErr(w, err)
		return
	}
	if selectAfter {
		if err := rt.UseDatabase(ctx, name); err != nil {
			writeErr(w, fmt.Errorf("restored but activate failed: %w", err))
			return
		}
		SetClientDBCookie(w, name)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": name})
}

func (rt *Runtime) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, err)
		return
	}
	if err := rt.Store.ChangeMasterPassword(body.OldPassword, body.NewPassword); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return &apiError{status: http.StatusBadRequest, msg: "invalid JSON: " + err.Error()}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	msg := err.Error()
	if ae, ok := err.(*apiError); ok {
		status = ae.status
		msg = ae.msg
	}
	writeJSON(w, status, map[string]any{"error": msg})
}
