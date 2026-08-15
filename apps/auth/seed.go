package auth

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

const defaultAdminPassword = "changeme"

func seedAdmin(ctx context.Context, svc *Service) error {
	ok, err := svc.HasAnyCredentials(ctx)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	email := strings.ToLower(strings.TrimSpace(os.Getenv("KaizenGo_ADMIN_EMAIL")))
	if email == "" {
		email = defaultAdminEmail
	}
	password := os.Getenv("KaizenGo_ADMIN_PASSWORD")
	if password == "" {
		password = defaultAdminPassword
		log.Printf("WARNING: using default admin password %q for %s — set KaizenGo_ADMIN_PASSWORD", defaultAdminPassword, email)
	}

	user, err := svc.FindUserByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("admin user %s not found (identity seed): %w", email, err)
	}
	if err := svc.SetPassword(ctx, user.ID, password); err != nil {
		return err
	}
	log.Printf("created default admin credentials for %s (org: %s)", email, user.OrgID)
	return nil
}

const defaultAdminEmail = "admin@kaizengo.local"
