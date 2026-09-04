package engine

import (
	"context"
	"testing"
)

func TestWithInternal(t *testing.T) {
	if IsInternal(context.Background()) {
		t.Fatal("plain context should not be internal")
	}
	if IsInternal(nil) {
		t.Fatal("nil context should not be internal")
	}
	ctx := WithInternal(context.Background())
	if !IsInternal(ctx) {
		t.Fatal("WithInternal context should be internal")
	}
	if !IsInternal(WithInternal(nil)) {
		t.Fatal("WithInternal(nil) should still mark the context")
	}
}
