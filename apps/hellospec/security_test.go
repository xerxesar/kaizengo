package hellospec

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"
)

func loadHelloSpecSecurity(t *testing.T) appspec.SecuritySpec {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	spec, err := appspec.LoadFile(filepath.Join(filepath.Dir(file), "app.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return spec.Security
}

func entriesFromSecurity(sec appspec.SecuritySpec) ([]acl.Entry, error) {
	out := make([]acl.Entry, 0, len(sec.Entries))
	for i, e := range sec.Entries {
		domainJSON := "[]"
		if e.Domain != nil {
			b, err := json.Marshal(e.Domain)
			if err != nil {
				return nil, err
			}
			domainJSON = string(b)
		}
		domain, err := acl.ParseDomain(domainJSON)
		if err != nil {
			return nil, fmt.Errorf("entry %s domain: %w", e.Name, err)
		}
		var fields []string
		if !e.Fields.All && len(e.Fields.Names) > 0 {
			fields = e.Fields.Names
		}
		effect := e.Effect
		if effect == "" {
			effect = acl.EffectAllow
		}
		out = append(out, acl.Entry{
			ID:       fmt.Sprint(i + 1),
			Effect:   effect,
			Resource: e.Resource,
			Actions:  e.Actions,
			Fields:   fields,
			Domain:   domain,
			Priority: e.Priority,
			Active:   true,
		})
	}
	return out, nil
}

func TestHelloSpecMemberACL(t *testing.T) {
	sec := loadHelloSpecSecurity(t)
	entries, err := entriesFromSecurity(sec)
	if err != nil {
		t.Fatal(err)
	}
	const greetingResource = "hellospec.greeting"
	p := acl.PrincipalContext{UserID: "jahan"}

	read := acl.Evaluate(entries, acl.Check{Resource: greetingResource, Action: acl.ActRead}, p)
	if !read.Allowed {
		t.Fatal("member should read")
	}
	denied := acl.DeniedFields(entries, greetingResource, acl.ActRead, p, nil)
	rec := acl.MaskRecord(map[string]any{
		"id": "1", "message": "Hi", "mood": "happy", "internalNote": "secret",
	}, read.FieldsRead, denied)
	if _, ok := rec["internalNote"]; ok {
		t.Fatal("internalNote must be masked for member")
	}
	if rec["message"] != "Hi" {
		t.Fatal("message should remain")
	}

	ownUpdate := acl.Evaluate(entries, acl.Check{
		Resource: greetingResource, Action: acl.ActUpdate,
		Record: map[string]any{"authorId": "jahan"},
		Fields: []string{"message"},
	}, p)
	if !ownUpdate.Allowed {
		t.Fatal("own update should allow")
	}

	otherUpdate := acl.Evaluate(entries, acl.Check{
		Resource: greetingResource, Action: acl.ActUpdate,
		Record: map[string]any{"authorId": "other"},
		Fields: []string{"message"},
	}, p)
	if otherUpdate.Allowed {
		t.Fatal("other's greeting must not be updatable")
	}

	noteWrite := acl.Evaluate(entries, acl.Check{
		Resource: greetingResource, Action: acl.ActUpdate,
		Record: map[string]any{"authorId": "jahan"},
		Fields: []string{"internalNote"},
	}, p)
	if noteWrite.Allowed {
		t.Fatal("internalNote write must deny")
	}

	ownNote := acl.Evaluate(entries, acl.Check{
		Resource: greetingResource, Action: acl.ActRead,
		Record: map[string]any{"authorId": "jahan"},
	}, p)
	if !ownNote.Allowed {
		t.Fatal("own read should allow")
	}
	ownDenied := acl.DeniedFields(entries, greetingResource, acl.ActRead, p, map[string]any{"authorId": "jahan"})
	ownRec := acl.MaskRecord(map[string]any{
		"id": "1", "authorId": "jahan", "message": "Hi", "mood": "happy", "internalNote": "mine",
	}, ownNote.FieldsRead, ownDenied)
	if ownRec["internalNote"] != "mine" {
		t.Fatalf("own internalNote should be visible, got %#v", ownRec)
	}

	otherNote := acl.Evaluate(entries, acl.Check{
		Resource: greetingResource, Action: acl.ActRead,
		Record: map[string]any{"authorId": "other"},
	}, p)
	otherDenied := acl.DeniedFields(entries, greetingResource, acl.ActRead, p, map[string]any{"authorId": "other"})
	otherRec := acl.MaskRecord(map[string]any{
		"id": "2", "authorId": "other", "message": "Yo", "mood": "happy", "internalNote": "theirs",
	}, otherNote.FieldsRead, otherDenied)
	if _, ok := otherRec["internalNote"]; ok {
		t.Fatal("other's internalNote must stay masked")
	}
}
