package views

type Kind string

const (
	ListView   Kind = "list"
	FormView   Kind = "form"
	KanbanView Kind = "kanban"
)

type Column struct {
	Key    string
	Label  string
	Width  string
	Align  string
	Hidden bool
}

type Field struct {
	Key      string
	Label    string
	Type     string
	Required bool
	Relation string
	Inverse  string
	Values   []string
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
	Name           string
	Model          string
	Kind           Kind
	Columns        []Column
	Fields         []Field
	FilterPresets  []FilterPreset
	ChartPresets   []ChartPreset
	ListQuery      string
	GetQuery       string
	CreateCommand  string
	UpdateCommand  string
	DeleteCommand  string
}
