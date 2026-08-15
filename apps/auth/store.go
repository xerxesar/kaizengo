package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists credentials and sessions.
type Store struct {
	pool   *pgxpool.Pool
	schema string
}

func NewStore(pool *pgxpool.Pool, schema string) *Store {
	if strings.TrimSpace(schema) == "" {
		schema = "auth"
	}
	return &Store{pool: pool, schema: schema}
}

func (s *Store) SetPassword(ctx context.Context, userID, passwordHash string) error {
	_, err := s.pool.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (user_id, password_hash, updated_at) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET password_hash = EXCLUDED.password_hash, updated_at = EXCLUDED.updated_at`,
		s.t("credentials"),
	), userID, passwordHash, time.Now().UTC())
	return err
}

func (s *Store) GetPasswordHash(ctx context.Context, userID string) (string, error) {
	var hash string
	err := s.pool.QueryRow(ctx, fmt.Sprintf(
		`SELECT password_hash FROM %s WHERE user_id = $1`, s.t("credentials"),
	), userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("no credentials")
	}
	return hash, err
}

func (s *Store) CreateSession(ctx context.Context, userID, orgID string, expiresAt time.Time) (*Session, error) {
	now := time.Now().UTC()
	sess := Session{
		ID: uuid.NewString(), UserID: userID, OrgID: orgID,
		ExpiresAt: expiresAt.UTC(), CreatedAt: now,
	}
	_, err := s.pool.Exec(ctx, fmt.Sprintf(
		`INSERT INTO %s (id, user_id, org_id, expires_at, created_at) VALUES ($1, $2, $3, $4, $5)`, s.t("sessions"),
	), sess.ID, sess.UserID, sess.OrgID, sess.ExpiresAt, sess.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx, fmt.Sprintf(
		`SELECT id, user_id, org_id, expires_at, created_at FROM %s WHERE id = $1`, s.t("sessions"),
	), sessionID).Scan(&sess.ID, &sess.UserID, &sess.OrgID, &sess.ExpiresAt, &sess.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("session not found")
	}
	return &sess, err
}

func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, s.t("sessions")), sessionID)
	return err
}

func (s *Store) HasAnyCredentials(ctx context.Context) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, s.t("credentials"))).Scan(&n)
	return n > 0, err
}

func (s *Store) t(name string) string {
	return `"` + strings.ReplaceAll(s.schema, `"`, `""`) + `"."` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
