package acl

import "testing"

func TestRegistryRegisterAndMerge(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ResourceDescriptor{
		App: "inventory", Kind: KindModel, Name: "product",
		Resource: "inventory.product", Label: "Product", Actions: CRUDActions(),
	})
	reg.RegisterOperation("appman", ActRead, "graphql", "apps")
	reg.RegisterOperation("appman", ActExecute, "graphql", "installApp")

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(all))
	}

	byApp := reg.ByApp("appman")
	if len(byApp) != 1 {
		t.Fatalf("expected 1 appman resource, got %d", len(byApp))
	}
	if len(byApp[0].Actions) != 2 {
		t.Fatalf("expected merged actions, got %v", byApp[0].Actions)
	}

	actions := reg.Actions()
	foundExecute := false
	for _, action := range actions {
		if action == ActExecute {
			foundExecute = true
		}
	}
	if !foundExecute {
		t.Fatalf("expected execute in actions, got %v", actions)
	}
}

func TestResourceHelpers(t *testing.T) {
	cases := map[string]string{
		ModelResource("inventory", "product"):     "inventory.product",
		MenuResource("permissions", "access"):     "permissions.menu.access",
		ViewResource("permissions", "Access"):     "permissions.view.Access",
		QueryResource("inventory", "inventoryViews"): "inventory.query.inventoryViews",
		MutationResource("appman", "installApp"):  "appman.mutation.installApp",
		EventResource("inventory", "stock_moved"): "inventory.event.stock_moved",
		NavResource("identity"):                   "identity.nav",
		AppResource("appman"):                     "appman",
	}
	for got, want := range cases {
		if got != want {
			t.Fatalf("resource helper: got %q want %q", got, want)
		}
	}
}

func TestMatchActionExecute(t *testing.T) {
	if !MatchAction([]string{ActExecute}, ActExecute) {
		t.Fatal("expected execute to match")
	}
	if MatchAction([]string{ActRead}, ActExecute) {
		t.Fatal("expected execute not to match read-only grant")
	}
}
