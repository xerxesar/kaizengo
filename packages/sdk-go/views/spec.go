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
}

type View struct {
	Name    string
	Model   string
	Kind    Kind
	Columns []Column
	Fields  []Field
}
