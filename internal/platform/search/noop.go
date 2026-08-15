package search

import "context"

type noopBackend struct{}

func (noopBackend) Query(_ context.Context, _, _ string, _ []string, _ int) ([]Hit, error) {
	return []Hit{}, nil
}

func (noopBackend) Upsert(_ context.Context, _ Document) error { return nil }

func (noopBackend) Delete(_ context.Context, _, _, _ string) error { return nil }
