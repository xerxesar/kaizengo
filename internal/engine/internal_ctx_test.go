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
	if !AllowsInternalWrite(ctx) {
		t.Fatal("WithInternal should allow internal writes")
	}
	write := WithInternalWrite(context.Background())
	if IsInternal(write) {
		t.Fatal("WithInternalWrite should not skip ACL")
	}
	if !AllowsInternalWrite(write) {
		t.Fatal("WithInternalWrite should allow internal writes")
	}
}
