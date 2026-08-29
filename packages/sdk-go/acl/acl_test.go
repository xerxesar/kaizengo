package acl

import (
	"testing"
)

func TestParseDomain(t *testing.T) {
	d, err := ParseDomain(`[["authorId","=","$user.id"],["status","=","draft"]]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 2 || d[0].Field != "authorId" || d[0].Op != "=" {
		t.Fatalf("unexpected domain: %+v", d)
	}
	p := PrincipalContext{UserID: "u1"}
	if !d.Match(map[string]any{"authorId": "u1", "status": "draft"}, p) {
		t.Fatal("expected match")
	}
	if d.Match(map[string]any{"authorId": "u2", "status": "draft"}, p) {
		t.Fatal("expected no match")
	}
}

func TestMatchResource(t *testing.T) {
	cases := []struct {
		pattern, want string
		ok            bool
	}{
		{"*", "inventory.product", true},
		{"inventory.*", "inventory.product", true},
		{"inventory.*", "identity.user", false},
		{"inventory.product", "inventory.product", true},
		{"inventory.product", "inventory.location", false},
	}
	for _, c := range cases {
		if MatchResource(c.pattern, c.want) != c.ok {
			t.Fatalf("MatchResource(%q,%q) want %v", c.pattern, c.want, c.ok)
		}
	}
}

func TestEvaluateCatalogDenyWithoutAllow(t *testing.T) {
	entries := []Entry{{
		ID: "1", Effect: EffectDeny, Resource: "identity.menu.users", Actions: []string{ActRead},
		Fields: []string{FieldsAll}, Priority: 2000, Active: true,
	}}
	p := PrincipalContext{Roles: []string{"member"}}

	deny := EvaluateCatalog(entries, Check{Resource: "identity.menu.users", Action: ActRead}, p)
	if deny.Allowed {
		t.Fatal("expected users menu denied")
	}
	open := EvaluateCatalog(entries, Check{Resource: "identity.menu.overview", Action: ActRead}, p)
	if !open.Allowed {
		t.Fatal("expected overview menu visible when no policy matches")
	}
}

func TestEvaluateAdminStar(t *testing.T) {
	entries := []Entry{{
		ID: "1", Effect: EffectAllow, Resource: "*", Actions: []string{"*"},
		Fields: nil, Priority: 0, Active: true,
	}}
	d := Evaluate(entries, Check{Resource: "inventory.product", Action: ActRead}, PrincipalContext{})
	if !d.Allowed || d.FieldsRead != nil {
		t.Fatalf("admin decision: %+v", d)
	}
}

func TestEvaluateFieldDenyOverride(t *testing.T) {
	entries := []Entry{
		{ID: "1", Effect: EffectAllow, Resource: "inventory.product", Actions: []string{ActRead},
			Fields: nil, Priority: 0, Active: true},
		{ID: "2", Effect: EffectDeny, Resource: "inventory.product", Actions: []string{ActRead},
			Fields: []string{"cost"}, Priority: 1000, Active: true},
	}
	p := PrincipalContext{UserID: "u1"}
	d := Evaluate(entries, Check{Resource: "inventory.product", Action: ActRead}, p)
	if !d.Allowed {
		t.Fatal("expected allow with field deny")
	}
	denied := DeniedFields(entries, "inventory.product", ActRead, p, nil)
	if len(denied) != 1 || denied[0] != "cost" {
		t.Fatalf("denied=%v", denied)
	}
	rec := MaskRecord(map[string]any{"name": "A", "cost": 9, "id": "1"}, d.FieldsRead, denied)
	if _, ok := rec["cost"]; ok {
		t.Fatal("cost should be masked")
	}
	if rec["name"] != "A" {
		t.Fatal("name should remain")
	}
}

func TestEvaluateWriteFields(t *testing.T) {
	entries := []Entry{{
		ID: "1", Effect: EffectAllow, Resource: "inventory.product", Actions: []string{ActUpdate},
		Fields: []string{"name", "sku"}, Priority: 0, Active: true,
	}}
	d := Evaluate(entries, Check{
		Resource: "inventory.product", Action: ActUpdate, Fields: []string{"name"},
	}, PrincipalContext{})
	if !d.Allowed {
		t.Fatal("name update should allow")
	}
	d2 := Evaluate(entries, Check{
		Resource: "inventory.product", Action: ActUpdate, Fields: []string{"cost"},
	}, PrincipalContext{})
	if d2.Allowed {
		t.Fatal("cost update should deny")
	}
}

func TestEvaluateRecordDomain(t *testing.T) {
	dom, _ := ParseDomain(`[["authorId","=","$user.id"]]`)
	entries := []Entry{{
		ID: "1", Effect: EffectAllow, Resource: "hellospec.greeting", Actions: []string{ActUpdate},
		Domain: dom, Priority: 0, Active: true,
	}}
	p := PrincipalContext{UserID: "u1"}
	ok := Evaluate(entries, Check{
		Resource: "hellospec.greeting", Action: ActUpdate,
		Record: map[string]any{"authorId": "u1"},
	}, p)
	if !ok.Allowed {
		t.Fatal("own record should allow")
	}
	no := Evaluate(entries, Check{
		Resource: "hellospec.greeting", Action: ActUpdate,
		Record: map[string]any{"authorId": "u2"},
	}, p)
	if no.Allowed {
		t.Fatal("other record should deny")
	}
}

func TestBuildListFilter(t *testing.T) {
	dom, _ := ParseDomain(`[["authorId","=","$user.id"]]`)
	entries := []Entry{{
		ID: "1", Effect: EffectAllow, Resource: "hellospec.greeting", Actions: []string{ActRead},
		Domain: dom, Priority: 0, Active: true,
	}}
	f := BuildListFilter(entries, "hellospec.greeting", ActRead, PrincipalContext{UserID: "u1"})
	if f.DenyAll || f.Unrestricted || len(f.Allow) != 1 {
		t.Fatalf("filter=%+v", f)
	}
}

func TestCompileListFilterSQL(t *testing.T) {
	dom, _ := ParseDomain(`[["authorId","=","$user.id"]]`)
	f := ListFilter{Allow: []Domain{dom}}
	col := func(field string) (string, bool) {
		if field == "authorId" {
			return `"author_id"`, true
		}
		return "", false
	}
	frag, err := CompileListFilter(f, col, 2, PrincipalContext{UserID: "u1"})
	if err != nil {
		t.Fatal(err)
	}
	if frag.Clause == "" || len(frag.Args) != 1 || frag.Args[0] != "u1" {
		t.Fatalf("frag=%+v", frag)
	}
}
