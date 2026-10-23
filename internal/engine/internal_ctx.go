package engine

import "context"

type internalCtxKey struct{}
type internalWriteCtxKey struct{}

// WithInternal marks ctx as a trusted in-process call (seeds, system jobs).
// It allows writes to internal models and skips model ACL.
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

// WithInternalWrite allows writes to internal: true models while keeping ACL.
// Used by CQRS command shorthand so field/domain rules still apply.
func WithInternalWrite(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, internalWriteCtxKey{}, true)
}

// AllowsInternalWrite reports whether ctx may write internal models.
func AllowsInternalWrite(ctx context.Context) bool {
	if IsInternal(ctx) {
		return true
	}
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(internalWriteCtxKey{}).(bool)
	return v
}
