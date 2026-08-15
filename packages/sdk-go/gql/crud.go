package gql

import "github.com/graphql-go/graphql"

type CRUDSpec struct {
	ListName   string
	GetName    string
	CreateName string
	UpdateName string
	DeleteName string

	ListField   *graphql.Field
	GetField    *graphql.Field
	CreateField *graphql.Field
	UpdateField *graphql.Field
	DeleteField *graphql.Field
}

type Registry interface {
	RegisterQuery(name string, field *graphql.Field)
	RegisterMutation(name string, field *graphql.Field)
}

func RegisterCRUD(reg Registry, spec CRUDSpec) {
	if spec.ListName != "" && spec.ListField != nil {
		reg.RegisterQuery(spec.ListName, spec.ListField)
	}
	if spec.GetName != "" && spec.GetField != nil {
		reg.RegisterQuery(spec.GetName, spec.GetField)
	}
	if spec.CreateName != "" && spec.CreateField != nil {
		reg.RegisterMutation(spec.CreateName, spec.CreateField)
	}
	if spec.UpdateName != "" && spec.UpdateField != nil {
		reg.RegisterMutation(spec.UpdateName, spec.UpdateField)
	}
	if spec.DeleteName != "" && spec.DeleteField != nil {
		reg.RegisterMutation(spec.DeleteName, spec.DeleteField)
	}
}
