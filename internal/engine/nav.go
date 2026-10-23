package engine

import (
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/internal/extension"
	"kaizengo/packages/sdk-go/views"
)

func menuCatalog(spec appspec.AppSpec) []views.Menu {
	return extension.BuildMenuCatalog(spec.Name, spec.Menus)
}
