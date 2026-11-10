package views

type Kind string

const (
	ListView   Kind = "list"
	FormView   Kind = "form"
	KanbanView Kind = "kanban"
	ScreenView Kind = "screen"
)

type Column struct {
	Key    string
	Label  string
	Width  string
	Align  string
	Hidden bool
	Widget string
	Span   int
	Order  int
	Group  string
}

type Field struct {
	Key      string
	Label    string
	Type     string
	Required bool
	Relation string
	Inverse  string
	Values   []string
	Span     int
	Widget   string
	Group    string
	Order    int
	Width    string
	Align    string
}

// ViewGroup is one form section from a spec screen.
type ViewGroup struct {
	ID       string
	Label    string
	LabelKey string
	Columns  int
	Order    int
}

type FilterPreset struct {
	ID       string
	Label    string
	LabelKey string
	Domain   string
	Q        string
	SearchIn []string
	GroupBy  []string
}

// ChartPreset is a named chart config from app.yaml models[].charts.
type ChartPreset struct {
	ID          string
	Label       string
	LabelKey    string
	Type        string
	Types       []string
	XField      string
	YField      string
	SeriesField string
	Measure     string
}

type View struct {
	Name          string
	Model         string
	Kind          Kind
	Columns       []Column
	Fields        []Field
	FilterPresets []FilterPreset
	ChartPresets  []ChartPreset
	ListQuery     string
	GetQuery      string
	CreateCommand string
	UpdateCommand string
	DeleteCommand string
	DefaultView   string
	Views         []string
	GroupBy       []string
	Groups        []ViewGroup
	// Layout is the nested region tree for a screen view.
	Layout *LayoutNode
	// Components are ids of child layouts ("{screen}.{id}") on a screen view.
	Components []string
}

// LayoutNode is one region: a component, or a row/column of nested regions.
type LayoutNode struct {
	Direction string
	Component string
	Size      string
	Span      int
	Items     []LayoutNode
}
