package acl

import "testing"

func TestParseDomainExprANDOR(t *testing.T) {
	expr, err := ParseDomainExpr(`["|",["mood","=","happy"],["mood","=","formal"]]`)
	if err != nil {
		t.Fatal(err)
	}
	if expr == nil || expr.Kind != "or" || len(expr.Children) != 2 {
		t.Fatalf("got %+v", expr)
	}

	flat, err := ParseDomainExpr(`[["message","ilike","%hi%"],["mood","=","happy"]]`)
	if err != nil {
		t.Fatal(err)
	}
	if flat.Kind != "and" || len(flat.Children) != 2 {
		t.Fatalf("flat and: %+v", flat)
	}

	rec := map[string]any{"mood": "happy", "message": "say hi there"}
	if !expr.Match(rec, PrincipalContext{}) {
		t.Fatal("or should match happy")
	}
	if !flat.Match(rec, PrincipalContext{}) {
		t.Fatal("and ilike should match")
	}
	rec2 := map[string]any{"mood": "neutral", "message": "hello"}
	if flat.Match(rec2, PrincipalContext{}) {
		t.Fatal("and should not match mood")
	}
}

func TestParseDomainExprNOT(t *testing.T) {
	expr, err := ParseDomainExpr(`["!",["mood","=","happy"]]`)
	if err != nil {
		t.Fatal(err)
	}
	if expr.Match(map[string]any{"mood": "happy"}, PrincipalContext{}) {
		t.Fatal("not should reject")
	}
	if !expr.Match(map[string]any{"mood": "formal"}, PrincipalContext{}) {
		t.Fatal("not should accept")
	}
}
