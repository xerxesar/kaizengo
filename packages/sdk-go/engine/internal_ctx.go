package engine

import "context"

type internalCtxKey struct{}

// WithInternal marks ctx as an in-process write. Models with internal: true
// accept create/update/delete only on this context (GraphQL stays read-only).
func WithInternal(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, internalCtxKey{}, true)
}

// IsInternal reports whether ctx was produced by WithInternal.
func IsInternal(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(internalCtxKey{}).(bool)
	return v
}
