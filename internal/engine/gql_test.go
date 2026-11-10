package engine

import (
	"testing"

	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/views"
)

func TestViewCatalogIncludesFilterPresets(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "appman",
		Models: []appspec.ModelSpec{{
			Name: "app",
			Fields: []appspec.FieldSpec{
				{Name: "status", Type: "enum", Values: []string{"installed", "available"}},
			},
			Filters: []appspec.FilterPresetSpec{{
				ID:       "installed",
				LabelKey: "appman.filter.installed",
				Domain:   [][]any{{"status", "=", "installed"}},
			}},
			Charts: []appspec.ChartPresetSpec{{
				ID:       "by_status",
				LabelKey: "appman.chart.by_status",
				Type:     "bar",
				Types:    []string{"bar", "pie"},
				XField:   "status",
				Measure:  "count",
			}},
		}},
	}
	list := buildListView(spec, spec.Models[0])
	if len(list.FilterPresets) != 1 || list.FilterPresets[0].ID != "installed" {
		t.Fatalf("unexpected filter presets: %+v", list.FilterPresets)
	}
	if len(list.ChartPresets) != 1 || list.ChartPresets[0].ID != "by_status" {
		t.Fatalf("unexpected chart presets: %+v", list.ChartPresets)
	}
	if list.ChartPresets[0].Type != "bar" || len(list.ChartPresets[0].Types) != 2 {
		t.Fatalf("unexpected chart preset fields: %+v", list.ChartPresets[0])
	}
}

func TestViewCatalogGeneratesListAndFormFromModels(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "hellospec",
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

func TestViewCatalogDeclaredReplaceAndAppend(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "hellospec",
		Models: []appspec.ModelSpec{{
			Name:   "greeting",
			Fields: []appspec.FieldSpec{{Name: "message", Type: "string", Required: true}},
		}},
		Views: []appspec.ViewSpec{
			{
				Name: "GreetingList", Kind: "collection", Model: "greeting",
				DefaultView: "kanban", Views: []string{"table", "kanban"}, GroupBy: []string{"message"},
				Columns: []appspec.ViewColumnSpec{{Field: "message", Width: "2fr", Widget: "textarea", Order: 1}},
			},
			{
				Name: "GreetingBoard", Kind: "collection", Model: "greeting",
				Columns: []appspec.ViewColumnSpec{{Field: "message", Order: 2}, {Field: "updatedAt", Order: 1}},
			},
			{Name: "GreetingCustom", Path: "views/GreetingCustom.page.tsx", Type: "page"},
		},
	}
	catalog := viewCatalog(spec)
	if len(catalog) != 3 {
		t.Fatalf("expected generated form plus two dynamic views (path ignored), got %d", len(catalog))
	}
	list := catalog[0]
	if list.Name != "GreetingList" || list.DefaultView != "kanban" || list.Columns[0].Width != "2fr" || list.Columns[0].Widget != "textarea" {
		t.Fatalf("view did not replace generated list: %+v", list)
	}
	board := catalog[2]
	if board.Name != "GreetingBoard" || board.Kind != views.ListView {
		t.Fatalf("expected appended board, got %+v", board)
	}
	if board.Columns[0].Key != "updatedAt" || board.Columns[1].Key != "message" {
		t.Fatalf("column order: %+v", board.Columns)
	}
}

func TestViewCatalogPageKeepsLayout(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "hellospec",
		Models: []appspec.ModelSpec{{
			Name:   "greeting",
			Fields: []appspec.FieldSpec{{Name: "message", Type: "string"}, {Name: "mood", Type: "string"}},
			Filters: []appspec.FilterPresetSpec{{
				ID: "all", Label: "All", Domain: `[]`,
			}},
		}},
		Views: []appspec.ViewSpec{{
			Name: "GreetingBoard", Path: "views/GreetingBoard.page.tsx",
			Kind: "collection", Model: "greeting", GroupBy: []string{"mood"},
			Columns: []appspec.ViewColumnSpec{{Field: "message", Order: 1}, {Field: "mood", Order: 2}},
			Filters: []appspec.FilterPresetSpec{{
				ID: "happy", Label: "Happy", Domain: `[["mood","=","happy"]]`,
			}},
		}},
	}
	catalog := viewCatalog(spec)
	var board *views.View
	for i := range catalog {
		if catalog[i].Name == "GreetingBoard" {
			board = &catalog[i]
			break
		}
	}
	if board == nil {
		t.Fatal("path page layout was not published")
	}
	if board.Columns[0].Key != "message" || len(board.GroupBy) != 1 || board.GroupBy[0] != "mood" {
		t.Fatalf("layout = %+v", board)
	}
	if len(board.FilterPresets) != 1 || board.FilterPresets[0].ID != "happy" {
		t.Fatalf("filters = %+v", board.FilterPresets)
	}
}

func TestViewCatalogPageComponents(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "hellospec",
		Models: []appspec.ModelSpec{{
			Name:   "greeting",
			Fields: []appspec.FieldSpec{{Name: "message", Type: "string"}, {Name: "mood", Type: "string"}},
		}},
		Views: []appspec.ViewSpec{{
			Name: "GreetingBoard", Path: "views/GreetingBoard.page.tsx",
			Components: []appspec.ViewComponentSpec{
				{ID: "board", Kind: "collection", Model: "greeting", GroupBy: []string{"mood"},
					Columns: []appspec.ViewColumnSpec{{Field: "message"}}},
				{ID: "composer", Kind: "form", Model: "greeting",
					Groups: []appspec.ViewGroupSpec{{Label: "New", Fields: []appspec.ViewColumnSpec{{Field: "message"}}}}},
			},
		}},
	}
	catalog := viewCatalog(spec)
	names := map[string]views.Kind{}
	for _, v := range catalog {
		names[v.Name] = v.Kind
	}
	if names["GreetingBoard.board"] != views.ListView || names["GreetingBoard.composer"] != views.FormView {
		t.Fatalf("catalog names = %+v", names)
	}
}

func TestViewCatalogDynamicScreen(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "hellospec",
		Models: []appspec.ModelSpec{{
			Name:   "greeting",
			Fields: []appspec.FieldSpec{{Name: "message", Type: "string"}},
		}},
		Views: []appspec.ViewSpec{{
			Name: "GreetingForm",
			Layout: appspec.ViewLayoutSpec{
				Direction: "column",
				Items: []appspec.ViewLayoutSpec{{
					Direction: "row",
					Items: []appspec.ViewLayoutSpec{
						{Component: "form", Span: 2},
						{Component: "recent", Span: 1, Size: "18rem"},
					},
				}},
			},
			Components: []appspec.ViewComponentSpec{
				{ID: "form", Kind: "form", Model: "greeting",
					Groups: []appspec.ViewGroupSpec{{Label: "New", Fields: []appspec.ViewColumnSpec{{Field: "message"}}}}},
				{ID: "recent", Kind: "collection", Model: "greeting",
					Columns: []appspec.ViewColumnSpec{{Field: "message"}}},
			},
		}},
	}
	catalog := viewCatalog(spec)
	var screen *views.View
	for i := range catalog {
		if catalog[i].Name == "GreetingForm" {
			screen = &catalog[i]
			break
		}
	}
	if screen == nil || screen.Kind != views.ScreenView || screen.Layout == nil {
		t.Fatalf("screen = %+v", screen)
	}
	row := screen.Layout.Items
	if screen.Layout.Direction != "column" || len(row) != 1 || row[0].Direction != "row" || len(row[0].Items) != 2 {
		t.Fatalf("layout = %+v", screen.Layout)
	}
	if row[0].Items[0].Component != "form" || row[0].Items[0].Span != 2 || row[0].Items[1].Size != "18rem" {
		t.Fatalf("regions = %+v", row[0].Items)
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

func TestViewCatalogSkipsFormForInternalModels(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "inventory",
		Models: []appspec.ModelSpec{{
			Name:     "cost_layer",
			Internal: true,
			Fields:   []appspec.FieldSpec{{Name: "quantity", Type: "number"}},
		}},
	}
	catalog := viewCatalog(spec)
	if len(catalog) != 1 {
		t.Fatalf("expected list-only catalog, got %d views: %+v", len(catalog), catalog)
	}
	if catalog[0].Kind != views.ListView || catalog[0].Model != "cost_layer" {
		t.Fatalf("unexpected view: %+v", catalog[0])
	}
}

func TestViewCatalogCQRSIncludesFormAndBindings(t *testing.T) {
	spec := appspec.AppSpec{
		Name: "hellospec",
		Models: []appspec.ModelSpec{{
			Name:     "greeting",
			Internal: true,
			Fields:   []appspec.FieldSpec{{Name: "message", Type: "string", Required: true}},
		}},
		Queries: []appspec.QuerySpec{
			{Name: "greetings", List: "greeting"},
			{Name: "greeting", Get: "greeting"},
		},
		Commands: []appspec.CommandSpec{
			{Name: "postGreeting", Create: "greeting"},
			{Name: "reviseGreeting", Update: "greeting"},
			{Name: "discardGreeting", Delete: "greeting"},
		},
	}
	catalog := viewCatalog(spec)
	if len(catalog) != 2 {
		t.Fatalf("expected list+form for CQRS internal model, got %d", len(catalog))
	}
	if catalog[0].ListQuery != "hellospecGreetings" || catalog[0].DeleteCommand != "hellospecDiscardGreeting" {
		t.Fatalf("unexpected list bindings: %+v", catalog[0])
	}
	if catalog[1].CreateCommand != "hellospecPostGreeting" || catalog[1].GetQuery != "hellospecGreeting" {
		t.Fatalf("unexpected form bindings: %+v", catalog[1])
	}
}
