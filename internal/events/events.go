package events

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrConcurrency = errors.New("event stream concurrency conflict")
	ErrNotFound    = errors.New("event stream not found")
)

type Event struct {
	ID         string          `json:"id"`
	StreamID   string          `json:"streamId"`
	StreamType string          `json:"streamType"`
	Version    int64           `json:"version"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	OccurredAt time.Time       `json:"occurredAt"`
}

type NewEvent struct {
	Type     string
	Payload  any
	Metadata any
}

type Appender interface {
	Append(ctx context.Context, streamID, streamType string, expectedVersion int64, events ...NewEvent) ([]Event, error)
}

type Loader interface {
	LoadStream(ctx context.Context, streamID string) ([]Event, error)
	LoadSince(ctx context.Context, fromID int64, limit int) ([]PersistedEvent, error)
}

type Store interface {
	Appender
	Loader
}

type PersistedEvent struct {
	RowID int64
	Event Event
}
