package views

// Menu is an in-app navigation item from app.yaml menus (plus addon contributions).
type Menu struct {
	ID        string
	Label     string
	LabelKey  string
	View      string
	Route     string
	Component string // optional addon component id (exports.menus)
	SourceApp string // addon that contributed this item (empty for local menus)
	Children  []Menu
}
