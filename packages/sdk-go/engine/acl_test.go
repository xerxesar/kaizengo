package engine

import (
	"testing"

	"kaizengo/packages/sdk-go/acl"
)

func TestModelResource(t *testing.T) {
	if modelResource("inventory", "product") != "inventory.product" {
		t.Fatal(modelResource("inventory", "product"))
	}
}

func TestRecordPassesFilter(t *testing.T) {
	dom, err := acl.ParseDomain(`[["authorId","=","$user.id"]]`)
	if err != nil {
		t.Fatal(err)
	}
	pc := acl.PrincipalContext{UserID: "u1"}
	f := acl.ListFilter{Allow: []acl.Domain{dom}}
	if !recordPassesFilter(Record{"authorId": "u1"}, f, pc) {
		t.Fatal("expected pass")
	}
	if recordPassesFilter(Record{"authorId": "u2"}, f, pc) {
		t.Fatal("expected fail")
	}
}
