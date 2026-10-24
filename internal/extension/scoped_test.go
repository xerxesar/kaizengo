package extension

import (
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestRegisterScopedRemove(t *testing.T) {
	calls := 0
	remove := RegisterScoped("model.scoped.item.beforeCreate", 0, func(Context) error {
		calls++
		return nil
	})
	ctx := Context{App: appspec.AppSpec{Name: "scoped"}}
	if err := Run("model.scoped.item.beforeCreate", ctx, true); err != nil {
		t.Fatal(err)
	}
	remove()
	remove()
	if err := Run("model.scoped.item.beforeCreate", ctx, true); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}
