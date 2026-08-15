package pgread

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SQLHandler struct {
	NameValue string
	Pool      *pgxpool.Pool
	SQL       string
	Args      func(any) []any
}

func (h *SQLHandler) Name() string { return h.NameValue }

func (h *SQLHandler) Exec(ctx context.Context, source any) error {
	if strings.TrimSpace(h.SQL) == "" {
		return fmt.Errorf("empty SQL")
	}
	_, err := h.Pool.Exec(ctx, h.SQL, h.Args(source)...)
	return err
}
