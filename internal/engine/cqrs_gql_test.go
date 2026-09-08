package engine

import (
	"testing"

	"kaizengo/packages/sdk-go/appspec"

	"github.com/graphql-go/graphql"
)

func TestGraphqlOutputFieldNamesFromRecordType(t *testing.T) {
	spec := appspec.AppSpec{Name: "hellospec"}
	svc := &modelService{model: appspec.ModelSpec{
		Name: "greeting",
		Fields: []appspec.FieldSpec{
			{Name: "message", Type: "string"},
			{Name: "mood", Type: "string"},
			{Name: "internalNote", Type: "string"},
		},
	}}
	obj := newRecordType(spec, svc)
	names := graphqlOutputFieldNames(graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(obj))))
	want := map[string]bool{
		"id": true, "orgId": true, "authorId": true, "deleted": true,
		"createdAt": true, "updatedAt": true,
		"message": true, "mood": true, "internalNote": true,
	}
	if len(names) != len(want) {
		t.Fatalf("got %v (len=%d), want %d names", names, len(names), len(want))
	}
	for _, n := range names {
		if !want[n] {
			t.Fatalf("unexpected field %q in %v", n, names)
		}
	}
}

func TestGraphqlArgNames(t *testing.T) {
	args := graphql.FieldConfigArgument{
		"id":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.ID)},
		"message": &graphql.ArgumentConfig{Type: graphql.String},
		"mood":    &graphql.ArgumentConfig{Type: graphql.String},
	}
	got := graphqlArgNames(args)
	if len(got) != 3 || got[0] != "id" || got[1] != "message" || got[2] != "mood" {
		t.Fatalf("got %v", got)
	}
}
