package appspec_test

import (
	"strings"
	"testing"

	"kaizengo/packages/sdk-go/appspec"
)

func TestMergeTwoWayKeepsOursOnlyAndTakesTheirsOnly(t *testing.T) {
	ours := appspec.AppSpec{
		Name: "crm", Title: "CRM", Summary: "Leads",
		Models: []appspec.ModelSpec{{
			Name: "lead",
			Fields: []appspec.FieldSpec{
				{Name: "title", Type: appspec.TypeString},
				{Name: "okok", Type: appspec.TypeString}, // user-only
			},
		}},
	}
	theirs := appspec.AppSpec{
		Name: "crm", Title: "CRM", Summary: "Leads",
		Models: []appspec.ModelSpec{{
			Name: "lead",
			Fields: []appspec.FieldSpec{
				{Name: "title", Type: appspec.TypeString},
				{Name: "stage", Type: appspec.TypeString}, // code-only
			},
		}},
	}
	got, err := appspec.MergeTwoWay(ours, theirs, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", got.Conflicts)
	}
	names := map[string]bool{}
	for _, f := range got.Spec.Models[0].Fields {
		names[f.Name] = true
	}
	if !names["okok"] || !names["stage"] || !names["title"] {
		t.Fatalf("fields = %+v", got.Spec.Models[0].Fields)
	}
}

func TestMergeTwoWayConflictOnDivergentField(t *testing.T) {
	ours := appspec.AppSpec{
		Name: "crm", Title: "CRM", Summary: "x",
		Models: []appspec.ModelSpec{{
			Name: "lead",
			Fields: []appspec.FieldSpec{{Name: "amount", Type: appspec.TypeNumber}},
		}},
	}
	theirs := appspec.AppSpec{
		Name: "crm", Title: "CRM", Summary: "x",
		Models: []appspec.ModelSpec{{
			Name: "lead",
			Fields: []appspec.FieldSpec{{Name: "amount", Type: appspec.TypeInt}},
		}},
	}
	got, err := appspec.MergeTwoWay(ours, theirs, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Conflicts) != 1 || !strings.Contains(got.Conflicts[0].Path, "amount") {
		t.Fatalf("conflicts = %+v", got.Conflicts)
	}
	if got.Spec.Models[0].Fields[0].Type != appspec.TypeNumber {
		t.Fatalf("ours should win until force, got %q", got.Spec.Models[0].Fields[0].Type)
	}
	forced, err := appspec.MergeTwoWay(ours, theirs, true)
	if err != nil || len(forced.Conflicts) != 0 {
		t.Fatalf("force: %+v err=%v", forced.Conflicts, err)
	}
	if forced.Spec.Models[0].Fields[0].Type != appspec.TypeInt {
		t.Fatalf("force should take theirs, got %q", forced.Spec.Models[0].Fields[0].Type)
	}
}

func TestMergeTwoWayViewConflict(t *testing.T) {
	ours := appspec.AppSpec{
		Name: "h", Title: "H", Summary: "s",
		Views: []appspec.ViewSpec{{Name: "GreetingForm", Kind: "form", Model: "greeting",
			Groups: []appspec.ViewGroupSpec{{Fields: []appspec.ViewColumnSpec{{Field: "message"}}}}}},
	}
	theirs := appspec.AppSpec{
		Name: "h", Title: "H", Summary: "s",
		Views: []appspec.ViewSpec{{Name: "GreetingForm", Kind: "form", Model: "greeting",
			Groups: []appspec.ViewGroupSpec{{Fields: []appspec.ViewColumnSpec{{Field: "message"}, {Field: "note"}}}}}},
	}
	got, err := appspec.MergeTwoWay(ours, theirs, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Conflicts) != 1 || got.Conflicts[0].Path != "views.GreetingForm" {
		t.Fatalf("conflicts = %+v", got.Conflicts)
	}
}

func TestMergeThreeWayAutoTakesCodeWhenTipUntouched(t *testing.T) {
	ancestor := appspec.AppSpec{
		Name: "h", Title: "H", Summary: "s",
		Models: []appspec.ModelSpec{{Name: "greeting", Fields: []appspec.FieldSpec{
			{Name: "message", Type: appspec.TypeString},
		}}},
		Views: []appspec.ViewSpec{{Name: "GreetingForm", Kind: "form", Model: "greeting",
			Groups: []appspec.ViewGroupSpec{{Fields: []appspec.ViewColumnSpec{{Field: "message"}}}}}},
	}
	// Tip: user added okok; form still as in v1 (ancestor).
	ours := appspec.AppSpec{
		Name: "h", Title: "H", Summary: "s",
		Models: []appspec.ModelSpec{{Name: "greeting", Fields: []appspec.FieldSpec{
			{Name: "message", Type: appspec.TypeString},
			{Name: "okok", Type: appspec.TypeString},
		}}},
		Views: []appspec.ViewSpec{{Name: "GreetingForm", Kind: "form", Model: "greeting",
			Groups: []appspec.ViewGroupSpec{{Fields: []appspec.ViewColumnSpec{{Field: "message"}}}}}},
	}
	// Code v2: changed the form, no okok.
	theirs := appspec.AppSpec{
		Name: "h", Title: "H", Summary: "s",
		Models: []appspec.ModelSpec{{Name: "greeting", Fields: []appspec.FieldSpec{
			{Name: "message", Type: appspec.TypeString},
		}}},
		Views: []appspec.ViewSpec{{Name: "GreetingForm", Kind: "form", Model: "greeting",
			Groups: []appspec.ViewGroupSpec{{Fields: []appspec.ViewColumnSpec{{Field: "message"}, {Field: "note"}}}}}},
	}

	got, err := appspec.MergeThreeWay(ancestor, ours, theirs, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Conflicts) != 0 {
		t.Fatalf("expected clean rebase, conflicts=%+v", got.Conflicts)
	}
	fields := map[string]bool{}
	for _, f := range got.Spec.Models[0].Fields {
		fields[f.Name] = true
	}
	if !fields["okok"] || !fields["message"] {
		t.Fatalf("fields = %+v", got.Spec.Models[0].Fields)
	}
	if len(got.Spec.Views[0].Groups[0].Fields) != 2 || got.Spec.Views[0].Groups[0].Fields[1].Field != "note" {
		t.Fatalf("view should take code: %+v", got.Spec.Views[0])
	}
}

func TestMergeThreeWayOptsResolvesConflict(t *testing.T) {
	ancestor := appspec.AppSpec{Name: "x", Title: "A"}
	ours := appspec.AppSpec{Name: "x", Title: "Ours"}
	theirs := appspec.AppSpec{Name: "x", Title: "Theirs"}
	got, err := appspec.MergeThreeWayOpts(ancestor, ours, theirs, appspec.MergeOpts{
		Resolve: map[string]string{"title": "theirs"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", got.Conflicts)
	}
	if got.Spec.Title != "Theirs" {
		t.Fatalf("title = %q", got.Spec.Title)
	}
}

func TestMergeThreeWayConflictsWhenBothChangedSamePath(t *testing.T) {
	ancestor := appspec.AppSpec{
		Name: "h", Title: "H", Summary: "s",
		Models: []appspec.ModelSpec{{Name: "greeting", Fields: []appspec.FieldSpec{
			{Name: "mood", Type: appspec.TypeString},
		}}},
	}
	ours := appspec.AppSpec{
		Name: "h", Title: "H", Summary: "s",
		Models: []appspec.ModelSpec{{Name: "greeting", Fields: []appspec.FieldSpec{
			{Name: "mood", Type: appspec.TypeString, Required: true},
		}}},
	}
	theirs := appspec.AppSpec{
		Name: "h", Title: "H", Summary: "s",
		Models: []appspec.ModelSpec{{Name: "greeting", Fields: []appspec.FieldSpec{
			{Name: "mood", Type: appspec.TypeString, Indexed: true},
		}}},
	}

	got, err := appspec.MergeThreeWay(ancestor, ours, theirs, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Conflicts) != 1 || !strings.Contains(got.Conflicts[0].Path, "mood") {
		t.Fatalf("conflicts = %+v", got.Conflicts)
	}
}
