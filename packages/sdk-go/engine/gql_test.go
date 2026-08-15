package engine

import (
	"testing"

	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/views"
)

func TestViewCatalogGeneratesListAndFormFromModels(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "hellospec",
		Views: []appspec.ViewSpec{
			{Name: "GreetingList", Type: "page"},
			{Name: "GreetingForm", Type: "page"},
		},
		Models: []appspec.ModelSpec{{
			Name: "greeting",
			Fields: []appspec.FieldSpec{{
				Name: "message", Type: "string", Required: true,
			}},
		}},
	}

	catalog := viewCatalog(spec)
	if len(catalog) != 2 {
		t.Fatalf("expected 2 model views, got %d", len(catalog))
	}

	list := catalog[0]
	if list.Name != "GreetingList" || list.Model != "greeting" || list.Kind != views.ListView {
		t.Fatalf("unexpected list view: %+v", list)
	}
	if len(list.Columns) != 2 || list.Columns[0].Key != "message" || list.Columns[1].Key != "updatedAt" {
		t.Fatalf("unexpected list columns: %+v", list.Columns)
	}

	form := catalog[1]
	if form.Name != "GreetingForm" || form.Model != "greeting" || form.Kind != views.FormView {
		t.Fatalf("unexpected form view: %+v", form)
	}
	if len(form.Fields) != 1 || form.Fields[0].Key != "message" || !form.Fields[0].Required {
		t.Fatalf("unexpected form fields: %+v", form.Fields)
	}
}

func TestViewCatalogIncludesRegisteredModels(t *testing.T) {
	t.Cleanup(func() {
		registry.mu.Lock()
		registry.byApp = map[string][]RegisteredModel{}
		registry.byName = map[string]map[string]RegisteredModel{}
		registry.mu.Unlock()
	})

	spec := appspec.AppSpec{Name: "identity", Resource: "identity"}
	registry.mu.Lock()
	registry.byName["identity"] = map[string]RegisteredModel{}
	registry.byApp["identity"] = []RegisteredModel{{
		Name: "user",
		Fields: []appspec.FieldSpec{
			{Name: "email", Type: "string", Required: true},
			{Name: "name", Type: "string", Required: true},
		},
		ListColumns: []views.Column{
			{Key: "name", Label: "Name"},
			{Key: "email", Label: "Email"},
		},
	}}
	registry.mu.Unlock()

	catalog := viewCatalog(spec)
	if len(catalog) != 2 {
		t.Fatalf("expected 2 registered views, got %d", len(catalog))
	}
	if catalog[0].Name != "UserList" || catalog[0].Model != "user" {
		t.Fatalf("unexpected list view: %+v", catalog[0])
	}
	if len(catalog[0].Columns) != 2 {
		t.Fatalf("expected custom list columns, got %+v", catalog[0].Columns)
	}
}

func TestViewCatalogMergesYamlAndRegistered(t *testing.T) {
	t.Cleanup(func() {
		registry.mu.Lock()
		registry.byApp = map[string][]RegisteredModel{}
		registry.byName = map[string]map[string]RegisteredModel{}
		registry.mu.Unlock()
	})

	spec := appspec.AppSpec{
		Name: "identity",
		Models: []appspec.ModelSpec{{
			Name: "user",
			Fields: []appspec.FieldSpec{
				{Name: "email", Type: "string", Required: true},
				{Name: "name", Type: "string", Required: true},
			},
		}},
	}
	registry.mu.Lock()
	registry.byName["identity"] = map[string]RegisteredModel{
		"user": {
			Name: "user",
			Fields: []appspec.FieldSpec{
				{Name: "email", Type: "string", Required: true},
				{Name: "name", Type: "string", Required: true},
			},
			ListColumns: []views.Column{
				{Key: "name", Label: "Name"},
				{Key: "email", Label: "Email"},
				{Key: "status", Label: "Status"},
			},
		},
	}
	registry.byApp["identity"] = []RegisteredModel{registry.byName["identity"]["user"]}
	registry.mu.Unlock()

	catalog := viewCatalog(spec)
	if len(catalog) != 2 {
		t.Fatalf("expected 2 views (no duplicates), got %d", len(catalog))
	}
	if len(catalog[0].Columns) != 3 || catalog[0].Columns[2].Key != "status" {
		t.Fatalf("expected registered ListColumns on yaml model, got %+v", catalog[0].Columns)
	}
}
