package mongoread

import (
	"context"
	"time"
)

// Adapter allows apps to plug any Mongo driver implementation while
// keeping sdk/projection free from concrete driver dependencies.
type Adapter interface {
	UpsertByID(ctx context.Context, id string, doc any) error
	DeleteByID(ctx context.Context, id string) error
}

type Sink struct {
	Adapter Adapter
	Timeout time.Duration
}

func (s Sink) UpsertByID(ctx context.Context, id string, doc any) error {
	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		defer cancel()
	}
	return s.Adapter.UpsertByID(ctx, id, doc)
}

func (s Sink) DeleteByID(ctx context.Context, id string) error {
	if s.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.Timeout)
		defer cancel()
	}
	return s.Adapter.DeleteByID(ctx, id)
}
