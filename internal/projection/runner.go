package projection

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kaizengo/packages/sdk-go/events"
)

var ErrNoEvents = errors.New("no events available")

type CheckpointStore interface {
	GetCheckpoint(ctx context.Context, name string) (int64, error)
	SaveCheckpoint(ctx context.Context, name string, rowID int64) error
}

type Handler interface {
	Name() string
	Handle(ctx context.Context, e events.PersistedEvent) error
}

type Runner struct {
	Name            string
	Store           events.Loader
	Checkpoints     CheckpointStore
	Handler         Handler
	BatchSize       int
	PollInterval    time.Duration
	StopOnEmptyPoll bool
}

func (r *Runner) Run(ctx context.Context) error {
	if r.Name == "" || r.Handler == nil || r.Store == nil || r.Checkpoints == nil {
		return fmt.Errorf("projection runner misconfigured")
	}
	if r.BatchSize <= 0 {
		r.BatchSize = 500
	}
	if r.PollInterval <= 0 {
		r.PollInterval = 250 * time.Millisecond
	}
	for {
		done, err := r.RunOnce(ctx)
		if err != nil {
			return err
		}
		if done && r.StopOnEmptyPoll {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.PollInterval):
		}
	}
}

// RunOnce processes one batch and returns true when nothing new was found.
func (r *Runner) RunOnce(ctx context.Context) (bool, error) {
	offset, err := r.Checkpoints.GetCheckpoint(ctx, r.Name)
	if err != nil {
		return false, err
	}
	batch, err := r.Store.LoadSince(ctx, offset, r.BatchSize)
	if err != nil {
		return false, err
	}
	if len(batch) == 0 {
		return true, nil
	}
	for _, item := range batch {
		if err := r.Handler.Handle(ctx, item); err != nil {
			return false, err
		}
		if err := r.Checkpoints.SaveCheckpoint(ctx, r.Name, item.RowID); err != nil {
			return false, err
		}
	}
	return false, nil
}
