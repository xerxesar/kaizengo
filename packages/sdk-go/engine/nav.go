package engine

import (
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/packages/sdk-go/extension"
	"kaizengo/packages/sdk-go/views"
)

func menuCatalog(spec appspec.AppSpec) []views.Menu {
	return extension.BuildMenuCatalog(spec.Name, spec.Menus)
}
