package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	idtypes "kaizengo/apps/identity/__types__"
	iauth "kaizengo/internal/auth"
	"kaizengo/internal/engine"

	"golang.org/x/crypto/bcrypt"
)

const Name = "auth"

const sessionDuration = 7 * 24 * time.Hour

type RoleLookup func(ctx context.Context, userID, orgID string) ([]string, error)

type Session struct {
	ID        string
	UserID    string
	OrgID     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type User struct {
	ID     string
	OrgID  string
	Email  string
	Name   string
	Status string
}

type AuthUser struct {
	ID    string   `json:"id"`
	OrgID string   `json:"orgId"`
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// Service handles login, logout, and session validation.
type Service struct {
	store *Store
	users *engine.ModelRegistry
	roles RoleLookup
}

func New(store *Store, users *engine.ModelRegistry, roles RoleLookup) *Service {
	return &Service{store: store, users: users, roles: roles}
}

func (s *Service) roleNames(ctx context.Context, userID, orgID string) []string {
	if s.roles == nil {
		return nil
	}
	names, _ := s.roles(ctx, userID, orgID)
	return names
}

func userFrom(rec engine.Record) User {
	return User{
		ID:     fmt.Sprint(rec["id"]),
		OrgID:  fmt.Sprint(rec["orgId"]),
		Email:  fmt.Sprint(rec["email"]),
		Name:   fmt.Sprint(rec["name"]),
		Status: fmt.Sprint(rec["status"]),
	}
}

func (s *Service) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	rec, err := s.users.FindBy(ctx, "user", "email", strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, err
	}
	u := userFrom(rec)
	return &u, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*Session, *AuthUser, error) {
	user, err := s.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid email or password")
	}
	if user.Status == string(idtypes.UserStatusSuspended) {
		return nil, nil, fmt.Errorf("account suspended")
	}
	hash, err := s.store.GetPasswordHash(ctx, user.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, nil, fmt.Errorf("invalid email or password")
	}
	sess, err := s.store.CreateSession(ctx, user.ID, user.OrgID, time.Now().UTC().Add(sessionDuration))
	if err != nil {
		return nil, nil, err
	}
	return sess, &AuthUser{
		ID: user.ID, OrgID: user.OrgID, Email: user.Email, Name: user.Name,
		Roles: s.roleNames(ctx, user.ID, user.OrgID),
	}, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

func (s *Service) ValidateSession(sessionID string) (*iauth.Principal, error) {
	ctx := context.Background()
	sess, err := s.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, iauth.ErrUnauthenticated
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_ = s.store.DeleteSession(ctx, sessionID)
		return nil, iauth.ErrUnauthenticated
	}
	rec, err := s.users.GetByID(ctx, "user", sess.UserID)
	if err != nil {
		return nil, iauth.ErrUnauthenticated
	}
	if fmt.Sprint(rec["status"]) == string(idtypes.UserStatusSuspended) {
		return nil, iauth.ErrUnauthenticated
	}
	return &iauth.Principal{
		UserID:    fmt.Sprint(rec["id"]),
		OrgID:     fmt.Sprint(rec["orgId"]),
		Email:     fmt.Sprint(rec["email"]),
		Name:      fmt.Sprint(rec["name"]),
		SessionID: sess.ID,
	}, nil
}

func (s *Service) Me(ctx context.Context) (*AuthUser, error) {
	p, err := iauth.MustPrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return &AuthUser{
		ID: p.UserID, OrgID: p.OrgID, Email: p.Email, Name: p.Name,
		Roles: s.roleNames(ctx, p.UserID, p.OrgID),
	}, nil
}

func (s *Service) SetPassword(ctx context.Context, userID, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.SetPassword(ctx, userID, string(hash))
}

func (s *Service) HasAnyCredentials(ctx context.Context) (bool, error) {
	return s.store.HasAnyCredentials(ctx)
}
