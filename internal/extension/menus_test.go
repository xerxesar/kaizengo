package extension

import (
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestBuildMenuCatalogMergesContributions(t *testing.T) {
	menuMu.Lock()
	prev := menuContributions
	menuContributions = nil
	menuMu.Unlock()
	t.Cleanup(func() {
		menuMu.Lock()
		menuContributions = prev
		menuMu.Unlock()
	})

	RegisterMenuContribution("typesense", appspec.MenuExtendSpec{
		App:       "settings",
		ID:        "search",
		LabelKey:  "typesense.menu.search",
		Label:     "Search",
		View:      "SearchSettings",
		Component: "typesense.SearchSettings",
		Order:     50,
	})
	RegisterMenuContribution("typesense", appspec.MenuExtendSpec{
		App:      "settings",
		Parent:   "general",
		ID:       "nested",
		Label:    "Nested",
		View:     "NestedView",
		Order:    10,
	})

	base := []appspec.MenuSpec{
		{ID: "general", LabelKey: "settings.menu.general", Label: "General", View: "General"},
	}
	got := BuildMenuCatalog("settings", base)
	if len(got) != 2 {
		t.Fatalf("top-level menus: got %d want 2 (general + search)", len(got))
	}
	if got[0].ID != "general" || got[1].ID != "search" {
		t.Fatalf("unexpected order/ids: %#v", got)
	}
	if got[1].Component != "typesense.SearchSettings" || got[1].SourceApp != "typesense" {
		t.Fatalf("search contribution fields: %#v", got[1])
	}
	if got[0].Route != "general" || got[1].Route != "search" {
		t.Fatalf("default routes from id: %#v / %#v", got[0].Route, got[1].Route)
	}
	if len(got[0].Children) != 1 || got[0].Children[0].ID != "nested" {
		t.Fatalf("expected nested under general, got %#v", got[0].Children)
	}
}
