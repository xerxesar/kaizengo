package auth

import (
	"encoding/json"
	"net/http"

	iauth "kaizengo/internal/auth"
)

type Handlers struct {
	Auth *Service
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	sess, user, err := h.Auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	iauth.SetSessionCookie(w, sess.ID, sess.ExpiresAt)
	writeJSON(w, http.StatusOK, user)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if p, ok := iauth.PrincipalFrom(r.Context()); ok {
		_ = h.Auth.Logout(r.Context(), p.SessionID)
	}
	iauth.ClearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.Auth.Me(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
