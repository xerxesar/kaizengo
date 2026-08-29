package acl

// CRUDActions are the standard data-model verbs.
func CRUDActions() []string {
	return []string{ActRead, ActCreate, ActUpdate, ActDelete}
}

// ReadActions is read-only access.
func ReadActions() []string {
	return []string{ActRead}
}

// ExecuteActions is operational / side-effect access without CRUD semantics.
func ExecuteActions() []string {
	return []string{ActExecute}
}

// CatalogActions covers read-only metadata surfaces (menus, views, catalogs).
func CatalogActions() []string {
	return []string{ActRead, ActExecute}
}

// AppActions covers app-level admin surfaces.
func AppActions() []string {
	return []string{ActRead, ActCreate, ActUpdate, ActDelete, ActExecute}
}

// StandardActions is the full platform action vocabulary for policy UIs.
func StandardActions() []string {
	return []string{ActRead, ActCreate, ActUpdate, ActDelete, ActExecute, ActAll}
}

func init() {
	for _, action := range StandardActions() {
		defaultRegistry.trackAction(action)
	}
}
