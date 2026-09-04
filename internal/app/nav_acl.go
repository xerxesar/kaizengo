package app

import (
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"
)

// RegisterShellNavResource catalogs the core Apps dropdown entry for ACL policies.
func RegisterShellNavResource(appName string, spec appspec.AppSpec) {
	if spec.Nav.Route == "" && spec.Nav.LabelKey == "" && spec.Nav.Label == "" {
		return
	}
	label := spec.Title
	if label == "" {
		label = appName
	}
	acl.Register(acl.ResourceDescriptor{
		App:         appName,
		Kind:        acl.KindNav,
		Name:        "shell",
		Resource:    acl.NavResource(appName),
		Label:       label,
		Description: "Shell Apps menu entry for " + appName,
		Actions:     acl.CatalogActions(),
		Surface:     "nav",
	})
}
