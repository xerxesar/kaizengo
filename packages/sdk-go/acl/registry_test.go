package acl

import "testing"

func TestRegistryRegisterAndMerge(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ResourceDescriptor{
		App: "inventory", Kind: KindModel, Name: "product",
		Resource: "inventory.product", Label: "Product", Actions: CRUDActions(),
	})
	// Models are never cataloged.
	if len(reg.All()) != 0 {
		t.Fatalf("expected models to be excluded from catalog, got %d", len(reg.All()))
	}

	reg.RegisterOperation("appman", ActRead, "graphql", "apps")
	reg.RegisterOperation("appman", ActExecute, "graphql", "installApp")

	all := reg.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(all))
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
		CommandResource("hellospec", "hellospecPostGreeting"): "hellospec.command.hellospecPostGreeting",
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

func TestEvaluateCallStyleIgnoresActions(t *testing.T) {
	entries := []Entry{{
		ID: "1", Effect: EffectAllow, Kind: KindCommand,
		Resource: "hellospec.command.hellospecPostGreeting",
		Actions:  []string{ActRead}, // wrong verb on purpose — must still match
		Fields:   []string{FieldsAll}, Priority: 0, Active: true,
	}}
	d := Evaluate(entries, Check{
		Resource: "hellospec.command.hellospecPostGreeting",
		Action:   ActExecute,
	}, PrincipalContext{})
	if !d.Allowed {
		t.Fatal("call-style grant should allow regardless of entry actions")
	}

	menu := []Entry{{
		ID: "2", Effect: EffectDeny, Kind: KindView, Resource: "identity.view.Users",
		Fields: []string{FieldsAll}, Priority: 0, Active: true,
	}}
	deny := EvaluateCatalog(menu, Check{Resource: "identity.view.Users", Action: ActRead}, PrincipalContext{})
	if deny.Allowed {
		t.Fatal("call-style deny should still apply")
	}
}

func TestRegisterPreservesFieldsAcrossOperation(t *testing.T) {
	reg := NewRegistry()
	reg.Register(ResourceDescriptor{
		App: "hellospec", Kind: KindQuery, Name: "hellospecGreeting",
		Resource: "hellospec.query.hellospecGreeting", Label: "greeting",
		Fields: []string{"message", "mood", "internalNote"}, Surface: "graphql",
	})
	reg.RegisterOperation("hellospec.query.hellospecGreeting", ActRead, "graphql", "hellospecGreeting")
	all := reg.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(all))
	}
	if len(all[0].Fields) != 3 {
		t.Fatalf("expected fields preserved, got %v", all[0].Fields)
	}
}

func TestInferKindCommand(t *testing.T) {
	if InferKind("hellospec.command.post") != KindCommand {
		t.Fatal("expected KindCommand")
	}
	if !IsCallStyle(KindQuery) || !IsCallStyle(KindCommand) || !IsCallStyle(KindView) {
		t.Fatal("call-style classification wrong")
	}
	if IsCallStyle(KindModel) || IsCallStyle(KindMenu) || IsCallStyle(KindNav) {
		t.Fatal("menu/nav/model should not be call-style catalog kinds")
	}
}
