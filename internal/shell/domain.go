package shell

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const domainUsage = "domain list [compiled|dynamic] | show <app> [--layer | --rev N] | history <app> | diff <<EOF … EOF | apply [--dry-run] <<EOF … EOF | add-field <app>.<model> <field> <type> [flags] | revert <app> <revision> [--dry-run] | delete <app> [--dry-run] | export [app …] | import [--dry-run] <<EOF … EOF"

const domainFields = `app kind title revision baseRevision layerRevision source loaded error spec layer`

const applyResultFields = `dryRun changes domain { app revision }`

func cmdDomain(s *session, args []string) (string, any, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("usage: %s", domainUsage)
	}
	rest, dryRun := dropFlag(args[1:], "--dry-run")
	switch args[0] {
	case "list", "ls":
		var vars map[string]any
		if len(rest) > 0 {
			vars = map[string]any{"kind": rest[0]}
		}
		data, err := s.ex.Exec(s.ctx, `query($kind:String){ platformDomains(kind:$kind) { app kind title revision layerRevision source loaded error } }`, vars)
		if err != nil {
			return "", nil, err
		}
		list, _ := data["platformDomains"].([]any)
		if len(list) == 0 {
			return "no domains (create one with: domain apply <<EOF … EOF)", list, nil
		}
		return formatValue(list, []string{"app", "kind", "title", "revision", "layerRevision", "source", "loaded", "error"}), list, nil
	case "show", "get":
		return domainShow(s, rest)
	case "history", "log":
		if len(rest) != 1 {
			return "", nil, fmt.Errorf("usage: domain history <app>")
		}
		data, err := s.ex.Exec(s.ctx, `query($app:String!){ platformDomainRevisions(app:$app) { revision layer source changes createdAt } }`, map[string]any{"app": rest[0]})
		if err != nil {
			return "", nil, err
		}
		list, _ := data["platformDomainRevisions"].([]any)
		if len(list) == 0 {
			return fmt.Sprintf("%s has no stored revisions", rest[0]), list, nil
		}
		rows := make([]any, 0, len(list))
		for _, raw := range list {
			r, _ := raw.(map[string]any)
			row := map[string]any{}
			for k, v := range r {
				row[k] = v
			}
			row["changes"] = joinAny(r["changes"], "; ")
			rows = append(rows, row)
		}
		return formatValue(rows, []string{"revision", "layer", "source", "createdAt", "changes"}), list, nil
	case "apply", "diff":
		if len(rest) != 1 {
			return "", nil, fmt.Errorf("usage: domain %s [--dry-run] <<EOF … EOF", args[0])
		}
		if args[0] == "diff" {
			data, err := s.ex.Exec(s.ctx, `query($s:String!){ platformDomainDiff(spec:$s) { app changes } }`, map[string]any{"s": rest[0]})
			if err != nil {
				return "", nil, err
			}
			d, _ := data["platformDomainDiff"].(map[string]any)
			return changeList(fmt.Sprintf("%v", d["app"]), d["changes"]), d, nil
		}
		return applyDomain(s, rest[0], dryRun)
	case "add-field":
		return domainAddField(s, rest, dryRun)
	case "revert":
		if len(rest) != 2 {
			return "", nil, fmt.Errorf("usage: domain revert <app> <revision> [--dry-run]")
		}
		rev, err := strconv.Atoi(strings.TrimPrefix(rest[1], "r"))
		if err != nil {
			return "", nil, fmt.Errorf("revision must be a number, got %q", rest[1])
		}
		data, err := s.ex.Exec(s.ctx, `mutation($app:String!,$r:Int!,$d:Boolean){ platformRevertDomain(app:$app, revision:$r, dryRun:$d) { `+applyResultFields+` } }`,
			map[string]any{"app": rest[0], "r": rev, "d": dryRun})
		if err != nil {
			return "", nil, err
		}
		return resultText(data["platformRevertDomain"], fmt.Sprintf("reverted %s to revision %d:", rest[0], rev))
	case "delete", "rm", "reset":
		if len(rest) != 1 {
			return "", nil, fmt.Errorf("usage: domain delete <app> [--dry-run]")
		}
		data, err := s.ex.Exec(s.ctx, `mutation($app:String!,$d:Boolean){ platformDeleteDomain(app:$app, dryRun:$d) { `+applyResultFields+` } }`,
			map[string]any{"app": rest[0], "d": dryRun})
		if err != nil {
			return "", nil, err
		}
		return resultText(data["platformDeleteDomain"], "removed layer")
	case "export", "backup":
		data, err := s.ex.Exec(s.ctx, `query($apps:[String!]){ platformExportDomains(apps:$apps) }`, map[string]any{"apps": rest})
		if err != nil {
			return "", nil, err
		}
		out := fmt.Sprint(data["platformExportDomains"])
		return strings.TrimRight(out, "\n"), out, nil
	case "import", "restore":
		if len(rest) != 1 {
			return "", nil, fmt.Errorf("usage: domain import [--dry-run] <<EOF … EOF")
		}
		data, err := s.ex.Exec(s.ctx, `mutation($b:String!,$d:Boolean){ platformImportDomains(bundle:$b, dryRun:$d) { `+applyResultFields+` } }`,
			map[string]any{"b": rest[0], "d": dryRun})
		if err != nil {
			return "", nil, err
		}
		list, _ := data["platformImportDomains"].([]any)
		if len(list) == 0 {
			return "bundle has no domains", list, nil
		}
		var parts []string
		for _, raw := range list {
			text, _, _ := resultText(raw, "restored")
			parts = append(parts, text)
		}
		return strings.Join(parts, "\n"), list, nil
	default:
		return "", nil, fmt.Errorf("unknown domain subcommand %q", args[0])
	}
}

func domainShow(s *session, args []string) (string, any, error) {
	args, layerOnly := dropFlag(args, "--layer")
	var rev string
	var pos []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--rev" && i+1 < len(args) {
			rev = args[i+1]
			i++
			continue
		}
		pos = append(pos, args[i])
	}
	if len(pos) != 1 {
		return "", nil, fmt.Errorf("usage: domain show <app> [--layer | --rev N]")
	}
	app := pos[0]
	if rev != "" {
		n, err := strconv.Atoi(strings.TrimPrefix(rev, "r"))
		if err != nil {
			return "", nil, fmt.Errorf("revision must be a number, got %q", rev)
		}
		data, err := s.ex.Exec(s.ctx, `query($app:String!,$r:Int!){ platformDomainRevision(app:$app, revision:$r) { revision layer source spec createdAt } }`,
			map[string]any{"app": app, "r": n})
		if err != nil {
			return "", nil, err
		}
		r, _ := data["platformDomainRevision"].(map[string]any)
		if r == nil {
			return "", nil, fmt.Errorf("%s has no revision %d", app, n)
		}
		header := fmt.Sprintf("# %s revision %v (%v layer, source %v, %v)", app, r["revision"], r["layer"], r["source"], r["createdAt"])
		body := strings.TrimRight(fmt.Sprint(r["spec"]), "\n")
		if body == "" {
			body = "# (empty layer)"
		}
		return header + "\n" + body, r, nil
	}
	d, err := fetchDomain(s, app)
	if err != nil {
		return "", nil, err
	}
	header := fmt.Sprintf("# %s %v revision %v (base r%v, layer r%v, source %v, loaded %v)",
		d["app"], d["kind"], d["revision"], d["baseRevision"], d["layerRevision"], d["source"], d["loaded"])
	if e, ok := d["error"].(string); ok && e != "" {
		header += "\n# error: " + e
	}
	body := fmt.Sprint(d["spec"])
	if layerOnly {
		header += "\n# stored layer only"
		body = fmt.Sprint(d["layer"])
		if strings.TrimSpace(body) == "" {
			body = "# (no layer: the base is unchanged)"
		}
	}
	return header + "\n" + strings.TrimRight(body, "\n"), d, nil
}

func fetchDomain(s *session, app string) (map[string]any, error) {
	data, err := s.ex.Exec(s.ctx, `query($app:String!){ platformDomain(app:$app) { `+domainFields+` } }`, map[string]any{"app": app})
	if err != nil {
		return nil, err
	}
	d, _ := data["platformDomain"].(map[string]any)
	if d == nil {
		return nil, fmt.Errorf("app %q not found", app)
	}
	return d, nil
}

func applyDomain(s *session, doc string, dryRun bool) (string, any, error) {
	data, err := s.ex.Exec(s.ctx, `mutation($s:String!,$d:Boolean){ platformApplyDomain(spec:$s, dryRun:$d) { `+applyResultFields+` } }`,
		map[string]any{"s": doc, "d": dryRun})
	if err != nil {
		return "", nil, err
	}
	return resultText(data["platformApplyDomain"], "applied")
}

func resultText(raw any, verb string) (string, any, error) {
	res, _ := raw.(map[string]any)
	dom, _ := res["domain"].(map[string]any)
	title := fmt.Sprintf("%s %v (revision %v)", verb, dom["app"], dom["revision"])
	if strings.HasSuffix(verb, ":") {
		title = fmt.Sprintf("%s now revision %v", verb, dom["revision"])
	}
	if dry, _ := res["dryRun"].(bool); dry {
		title = fmt.Sprintf("dry run for %v", dom["app"])
	}
	return changeList(title, res["changes"]), res, nil
}

func dropFlag(args []string, flag string) ([]string, bool) {
	out := make([]string, 0, len(args))
	found := false
	for _, a := range args {
		if a == flag {
			found = true
			continue
		}
		out = append(out, a)
	}
	return out, found
}

func joinAny(raw any, sep string) string {
	list, _ := raw.([]any)
	parts := make([]string, 0, len(list))
	for _, v := range list {
		parts = append(parts, fmt.Sprint(v))
	}
	return strings.Join(parts, sep)
}

// newField is one field added by `domain add-field`.
type newField struct {
	Name, Type, Relation        string
	Values                      []string
	Required, Indexed, Readonly bool
}

func domainAddField(s *session, args []string, dryRun bool) (string, any, error) {
	usage := fmt.Errorf("usage: domain add-field <app>.<model> <field> <type> [--required] [--indexed] [--readonly] [--values a,b] [--relation app.model] [--dry-run]")
	var pos []string
	var f newField
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--required":
			f.Required = true
		case "--indexed":
			f.Indexed = true
		case "--readonly":
			f.Readonly = true
		case "--values", "--relation":
			if i+1 >= len(args) {
				return "", nil, usage
			}
			i++
			if a == "--values" {
				f.Values = splitList(args[i])
			} else {
				f.Relation = args[i]
			}
		default:
			if strings.HasPrefix(a, "--") {
				return "", nil, fmt.Errorf("unknown flag %s", a)
			}
			pos = append(pos, a)
		}
	}
	if len(pos) != 3 {
		return "", nil, usage
	}
	app, model, ok := strings.Cut(pos[0], ".")
	if !ok || app == "" || model == "" {
		return "", nil, usage
	}
	f.Name, f.Type = pos[1], pos[2]

	d, err := fetchDomain(s, app)
	if err != nil {
		return "", nil, err
	}
	doc, err := addFieldToDoc(fmt.Sprint(d["spec"]), model, f)
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun)
}

// addFieldToDoc appends f to model's fields in an app.yaml document, keeping
// the rest of the document (order, comments) as written.
func addFieldToDoc(doc, model string, f newField) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return "", fmt.Errorf("parse domain: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("domain document is not a mapping")
	}
	models := mapValue(root.Content[0], "models")
	if models == nil || models.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("domain has no models list")
	}
	var target *yaml.Node
	for _, m := range models.Content {
		if n := mapValue(m, "name"); n != nil && n.Value == model {
			target = m
			break
		}
	}
	if target == nil {
		return "", fmt.Errorf("model %q not found", model)
	}
	fields := mapValue(target, "fields")
	if fields == nil {
		fields = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		target.Content = append(target.Content, scalar("fields"), fields)
	}
	for _, existing := range fields.Content {
		if n := mapValue(existing, "name"); n != nil && n.Value == f.Name {
			return "", fmt.Errorf("model %s already has field %q", model, f.Name)
		}
	}
	field := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	field.Content = append(field.Content, scalar("name"), scalar(f.Name), scalar("type"), scalar(f.Type))
	for _, flag := range []struct {
		key string
		on  bool
	}{{"required", f.Required}, {"indexed", f.Indexed}, {"readonly", f.Readonly}} {
		if flag.on {
			field.Content = append(field.Content, scalar(flag.key), &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
		}
	}
	if len(f.Values) > 0 {
		vals := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, v := range f.Values {
			vals.Content = append(vals.Content, scalar(v))
		}
		field.Content = append(field.Content, scalar("values"), vals)
	}
	if f.Relation != "" {
		field.Content = append(field.Content, scalar("relation"), scalar(f.Relation))
	}
	fields.Content = append(fields.Content, field)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func scalar(v string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
}
