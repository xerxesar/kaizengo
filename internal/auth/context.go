package auth

import "context"

type ctxKey struct{}

// Principal is the authenticated user attached to a request.
type Principal struct {
	UserID    string
	OrgID     string
	Email     string
	Name      string
	SessionID string
}

func WithPrincipal(ctx context.Context, p *Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

func PrincipalFrom(ctx context.Context) (*Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(*Principal)
	return p, ok && p != nil
}

func MustPrincipal(ctx context.Context) (*Principal, error) {
	p, ok := PrincipalFrom(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}
	return p, nil
}
