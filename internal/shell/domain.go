package shell

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const domainUsage = "domain list [compiled|dynamic] | show <app> [--rev N] | show-view|show-menu|show-query|show-command|show-model|show-policy <app> <name> | history <app> | conflicts <app> | rebase <app> [--force] [--dry-run] | diff|apply [<app>] [-ted] [<<EOF … EOF] | add-field|set-field|remove-field | add-index|set-index|remove-index | add-view|set-view|remove-view | add-menu|set-menu|remove-menu | add-query|set-query|remove-query | add-command|set-command|remove-command | revert <app> <revision> [--dry-run] | delete <app> [--dry-run] | export [app …] | import [--dry-run] <<EOF … EOF"

const domainFields = `app kind title revision source loaded error rebaseStatus conflictCount spec`

const applyResultFields = `dryRun changes migrateRequired proposedSpec migrateHints { path hint stub } domain { app revision rebaseStatus conflictCount spec }`

func cmdDomain(s *session, args []string) (string, any, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("usage: %s", domainUsage)
	}
	rest, dryRun := dropFlag(args[1:], "--dry-run")
	rest, force := dropFlag(rest, "--force")
	rest, noMigrate := dropFlag(rest, "--no-migrate")
	rest, ted := takeTedFlag(rest)
	rest, migratePre, _ := takeOptValue(rest, "--migrate-pre")
	rest, migratePost, _ := takeOptValue(rest, "--migrate-post")
	rest, migrateSQL, _ := takeOptValue(rest, "--migrate") // alias of --migrate-post
	if migratePost == "" {
		migratePost = migrateSQL
	}
	rest, migrateGo, _ := takeOptValue(rest, "--migrate-go")
	mig := migrateOpts{Pre: migratePre, Post: migratePost, Go: migrateGo, NoMigrate: noMigrate}
	switch args[0] {
	case "list", "ls":
		var vars map[string]any
		if len(rest) > 0 {
			vars = map[string]any{"kind": rest[0]}
		}
		data, err := s.ex.Exec(s.ctx, `query($kind:String){ platformDomains(kind:$kind) { app kind title revision source loaded error rebaseStatus conflictCount } }`, vars)
		if err != nil {
			return "", nil, err
		}
		list, _ := data["platformDomains"].([]any)
		if len(list) == 0 {
			return "no domains (create one with: domain apply <<EOF … EOF)", list, nil
		}
		return formatValue(list, []string{"app", "kind", "title", "revision", "source", "loaded", "rebaseStatus", "conflictCount", "error"}), list, nil
	case "show", "get":
		return domainShow(s, rest)
	case "show-view":
		return domainShowNamed(s, rest, "views", "name", "view")
	case "show-menu":
		return domainShowNamed(s, rest, "menus", "id", "menu")
	case "show-query":
		return domainShowNamed(s, rest, "queries", "name", "query")
	case "show-command":
		return domainShowNamed(s, rest, "commands", "name", "command")
	case "show-model":
		return domainShowNamed(s, rest, "models", "name", "model")
	case "show-policy":
		return domainShowPolicy(s, rest)
	case "conflicts":
		if len(rest) != 1 {
			return "", nil, fmt.Errorf("usage: domain conflicts <app>")
		}
		data, err := s.ex.Exec(s.ctx, `query($app:String!){ platformDomainConflicts(app:$app) { path ancestor ours theirs } }`, map[string]any{"app": rest[0]})
		if err != nil {
			return "", nil, err
		}
		list, _ := data["platformDomainConflicts"].([]any)
		if len(list) == 0 {
			return fmt.Sprintf("%s has no pending conflicts", rest[0]), list, nil
		}
		return formatValue(list, []string{"path", "ancestor", "ours", "theirs"}), list, nil
	case "rebase":
		if len(rest) != 1 {
			return "", nil, fmt.Errorf("usage: domain rebase <app> [--force] [--dry-run]")
		}
		data, err := s.ex.Exec(s.ctx, `mutation($app:String!,$f:Boolean,$d:Boolean){ platformRebaseDomain(app:$app, force:$f, dryRun:$d) { `+applyResultFields+` } }`,
			map[string]any{"app": rest[0], "f": force, "d": dryRun})
		if err != nil {
			return "", nil, err
		}
		return resultText(data["platformRebaseDomain"], "rebased")
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
		if ted {
			if len(rest) != 1 {
				return "", nil, fmt.Errorf("usage: domain %s <app> -ted", args[0])
			}
			d, err := fetchDomain(s, rest[0])
			if err != nil {
				return "", nil, err
			}
			seed := strings.TrimRight(fmt.Sprint(d["spec"]), "\n")
			return tedSoft(fmt.Sprintf("domain %s", args[0]), seed)
		}
		if len(rest) != 1 {
			return "", nil, fmt.Errorf("usage: domain %s [--dry-run] [--force] [--migrate-pre=SQL|--migrate=SQL|--migrate-go n|--no-migrate] <<EOF … EOF", args[0])
		}
		if args[0] == "diff" {
			data, err := s.ex.Exec(s.ctx, `query($s:String!){ platformDomainDiff(spec:$s) { app changes } }`, map[string]any{"s": rest[0]})
			if err != nil {
				return "", nil, err
			}
			d, _ := data["platformDomainDiff"].(map[string]any)
			return changeList(fmt.Sprintf("%v", d["app"]), d["changes"]), d, nil
		}
		return applyDomain(s, rest[0], dryRun, force, mig)
	case "add-field":
		return domainAddField(s, rest, dryRun, mig)
	case "set-field":
		return domainSetField(s, rest, dryRun, mig)
	case "remove-field":
		return domainRemoveField(s, rest, dryRun, force, mig)
	case "add-index":
		return domainAddIndex(s, rest, dryRun)
	case "set-index":
		return domainSetIndex(s, rest, dryRun)
	case "remove-index":
		return domainRemoveIndex(s, rest, dryRun, force)
	case "add-view", "add-screen":
		return domainAddView(s, rest, dryRun)
	case "set-view":
		return domainSetView(s, rest, dryRun, ted)
	case "remove-view":
		return domainRemoveNamed(s, rest, dryRun, false, "views", "view")
	case "add-menu":
		return domainAddMenu(s, rest, dryRun)
	case "set-menu":
		return domainSetMenu(s, rest, dryRun, ted)
	case "remove-menu":
		return domainRemoveNamed(s, rest, dryRun, false, "menus", "menu")
	case "add-query":
		return domainAddCQRS(s, rest, dryRun, "queries", "query", false, ted)
	case "set-query":
		return domainAddCQRS(s, rest, dryRun, "queries", "query", true, ted)
	case "remove-query":
		return domainRemoveNamed(s, rest, dryRun, false, "queries", "query")
	case "add-command":
		return domainAddCQRS(s, rest, dryRun, "commands", "command", false, ted)
	case "set-command":
		return domainAddCQRS(s, rest, dryRun, "commands", "command", true, ted)
	case "remove-command":
		return domainRemoveNamed(s, rest, dryRun, false, "commands", "command")
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
		return resultText(data["platformDeleteDomain"], "cleared tip")
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
	args, _ = dropFlag(args, "--layer") // deprecated; tip is always the full document
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
		return "", nil, fmt.Errorf("usage: domain show <app> [--rev N]")
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
		header := fmt.Sprintf("# %s revision %v (source %v, %v)", app, r["revision"], r["source"], r["createdAt"])
		body := strings.TrimRight(fmt.Sprint(r["spec"]), "\n")
		if body == "" {
			body = "# (empty tip)"
		}
		return header + "\n" + body, r, nil
	}
	d, err := fetchDomain(s, app)
	if err != nil {
		return "", nil, err
	}
	header := fmt.Sprintf("# %s %v revision %v (source %v, loaded %v)",
		d["app"], d["kind"], d["revision"], d["source"], d["loaded"])
	if rs, ok := d["rebaseStatus"].(string); ok && rs != "" {
		header += fmt.Sprintf("\n# rebase: %s (%v conflicts)", rs, d["conflictCount"])
	}
	if e, ok := d["error"].(string); ok && e != "" {
		header += "\n# error: " + e
	}
	body := fmt.Sprint(d["spec"])
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

type migrateOpts struct {
	Pre, Post, Go string
	NoMigrate     bool
}

func (m migrateOpts) hasExplicit() bool {
	return strings.TrimSpace(m.Pre) != "" || strings.TrimSpace(m.Post) != "" || strings.TrimSpace(m.Go) != "" || m.NoMigrate
}

func applyDomain(s *session, doc string, dryRun, force bool, mig migrateOpts) (string, any, error) {
	data, err := s.ex.Exec(s.ctx, `mutation($s:String!,$d:Boolean,$f:Boolean,$pre:String,$post:String,$g:String,$n:Boolean){ platformApplyDomain(spec:$s, dryRun:$d, force:$f, migratePre:$pre, migratePost:$post, migrateGo:$g, noMigrate:$n) { `+applyResultFields+` } }`,
		map[string]any{"s": doc, "d": dryRun, "f": force, "pre": mig.Pre, "post": mig.Post, "g": mig.Go, "n": mig.NoMigrate})
	if err != nil {
		return "", nil, err
	}
	res, _ := data["platformApplyDomain"].(map[string]any)
	if req, _ := res["migrateRequired"].(bool); req {
		var b strings.Builder
		b.WriteString("migration required — open Settings → Domains to resolve\n")
		if hints, ok := res["migrateHints"].([]any); ok {
			for _, raw := range hints {
				h, _ := raw.(map[string]any)
				b.WriteString(fmt.Sprintf("  - %v: %v\n", h["path"], h["hint"]))
			}
		}
		return strings.TrimRight(b.String(), "\n"), res, fmt.Errorf("migration required")
	}
	return resultText(res, "applied")
}

func takeOptValue(args []string, flag string) ([]string, string, bool) {
	out := make([]string, 0, len(args))
	val, ok := "", false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == flag {
			if i+1 >= len(args) {
				return out, "", true
			}
			val, ok = args[i+1], true
			i++
			continue
		}
		if strings.HasPrefix(a, flag+"=") {
			val, ok = strings.TrimPrefix(a, flag+"="), true
			continue
		}
		out = append(out, a)
	}
	return out, val, ok
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
	Name, Type, Relation, OnDelete string
	Compute, Related               string
	Values, Depends                []string
	Required, Indexed, Readonly    bool
	Unique                         bool
	Store                          *bool
}

func domainAddField(s *session, args []string, dryRun bool, mig migrateOpts) (string, any, error) {
	usage := fmt.Errorf("usage: domain add-field <app>.<model> <field> <type> [--required] [--indexed] [--unique] [--readonly] [--values a,b] [--relation app.model] [--on-delete restrict|setnull|cascade] [--compute expr] [--related path] [--depends a,b] [--store|--no-store] [--dry-run]")
	var pos []string
	var f newField
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--required":
			f.Required = true
		case "--indexed":
			f.Indexed = true
		case "--unique":
			f.Unique = true
		case "--readonly":
			f.Readonly = true
		case "--values", "--relation", "--on-delete", "--compute", "--related", "--depends":
			if i+1 >= len(args) {
				return "", nil, usage
			}
			i++
			switch a {
			case "--values":
				f.Values = splitList(args[i])
			case "--relation":
				f.Relation = args[i]
			case "--compute":
				f.Compute = args[i]
			case "--related":
				f.Related = args[i]
			case "--depends":
				f.Depends = splitList(args[i])
			default:
				f.OnDelete = args[i]
			}
		case "--store":
			stored := true
			f.Store = &stored
		case "--no-store":
			stored := false
			f.Store = &stored
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
	return applyDomain(s, doc, dryRun, false, mig)
}

func domainSetField(s *session, args []string, dryRun bool, mig migrateOpts) (string, any, error) {
	usage := fmt.Errorf("usage: domain set-field <app>.<model> <field> <type> [--name new] [--required] [--indexed] [--unique] [--readonly] [--values a,b] [--relation app.model] [--on-delete restrict|setnull|cascade] [--compute expr] [--related path] [--depends a,b] [--store|--no-store] [--dry-run]")
	var pos []string
	var f newField
	renameTo := ""
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--required":
			f.Required = true
		case "--indexed":
			f.Indexed = true
		case "--unique":
			f.Unique = true
		case "--readonly":
			f.Readonly = true
		case "--name", "--values", "--relation", "--on-delete", "--compute", "--related", "--depends":
			if i+1 >= len(args) {
				return "", nil, usage
			}
			i++
			switch a {
			case "--name":
				renameTo = args[i]
			case "--values":
				f.Values = splitList(args[i])
			case "--relation":
				f.Relation = args[i]
			case "--compute":
				f.Compute = args[i]
			case "--related":
				f.Related = args[i]
			case "--depends":
				f.Depends = splitList(args[i])
			default:
				f.OnDelete = args[i]
			}
		case "--store":
			stored := true
			f.Store = &stored
		case "--no-store":
			stored := false
			f.Store = &stored
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
	oldName := pos[1]
	f.Type = pos[2]
	f.Name = oldName
	if renameTo != "" {
		f.Name = renameTo
	}
	d, err := fetchDomain(s, app)
	if err != nil {
		return "", nil, err
	}
	doc, err := setFieldInDoc(fmt.Sprint(d["spec"]), model, oldName, f)
	if err != nil {
		return "", nil, err
	}
	if renameTo != "" && renameTo != oldName && !mig.hasExplicit() {
		mig.Pre = fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s;",
			quoteIdent(pluralizeModel(model)), quoteIdent(fieldColName(oldName)), quoteIdent(fieldColName(renameTo)))
	}
	return applyDomain(s, doc, dryRun, false, mig)
}

func domainRemoveField(s *session, args []string, dryRun, force bool, mig migrateOpts) (string, any, error) {
	usage := fmt.Errorf("usage: domain remove-field <app>.<model> <field> [--force] [--dry-run]")
	var pos []string
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			return "", nil, fmt.Errorf("unknown flag %s", a)
		}
		pos = append(pos, a)
	}
	if len(pos) != 2 {
		return "", nil, usage
	}
	app, model, ok := strings.Cut(pos[0], ".")
	if !ok || app == "" || model == "" {
		return "", nil, usage
	}
	d, err := fetchDomain(s, app)
	if err != nil {
		return "", nil, err
	}
	doc, err := removeFieldFromDoc(fmt.Sprint(d["spec"]), model, pos[1])
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, force, mig)
}

func appendComputeKeys(field *yaml.Node, f newField) {
	if f.Compute != "" {
		setMap(field, "compute", scalar(f.Compute))
	}
	if f.Related != "" {
		setMap(field, "related", scalar(f.Related))
	}
	if len(f.Depends) > 0 {
		vals := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, v := range f.Depends {
			vals.Content = append(vals.Content, scalar(v))
		}
		setMap(field, "depends", vals)
	}
	if f.Store != nil {
		val := "false"
		if *f.Store {
			val = "true"
		}
		setMap(field, "store", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: val})
	}
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
	}{{"required", f.Required}, {"indexed", f.Indexed}, {"unique", f.Unique}, {"readonly", f.Readonly}} {
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
	if f.OnDelete != "" {
		field.Content = append(field.Content, scalar("onDelete"), scalar(f.OnDelete))
	}
	appendComputeKeys(field, f)
	fields.Content = append(fields.Content, field)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func domainAddIndex(s *session, args []string, dryRun bool) (string, any, error) {
	usage := fmt.Errorf("usage: domain add-index <app>.<model> <field>[,field…] [--unique] [--name n] [--method btree|hash|gin] [--include a,b] [--where \"col = 'v'\"] [--dry-run]")
	var pos []string
	name, method, where := "", "btree", ""
	var include []string
	unique := false
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--unique":
			unique = true
		case "--name", "--method", "--include", "--where":
			if i+1 >= len(args) {
				return "", nil, usage
			}
			i++
			switch a {
			case "--name":
				name = args[i]
			case "--method":
				method = args[i]
			case "--include":
				include = splitList(args[i])
			default:
				where = args[i]
			}
		default:
			if strings.HasPrefix(a, "--") {
				return "", nil, fmt.Errorf("unknown flag %s", a)
			}
			pos = append(pos, a)
		}
	}
	if len(pos) != 2 {
		return "", nil, usage
	}
	app, model, ok := strings.Cut(pos[0], ".")
	if !ok || app == "" || model == "" || pos[1] == "" {
		return "", nil, usage
	}
	d, err := fetchDomain(s, app)
	if err != nil {
		return "", nil, err
	}
	doc, err := addIndexToDoc(fmt.Sprint(d["spec"]), model, name, method, unique, splitList(pos[1]), include, where)
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, false, migrateOpts{})
}

func addIndexToDoc(doc, model, name, method string, unique bool, fields, include []string, where string) (string, error) {
	root, target, err := modelNode(doc, model)
	if err != nil {
		return "", err
	}
	indexes := mapValue(target, "indexes")
	if indexes == nil {
		indexes = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		target.Content = append(target.Content, scalar("indexes"), indexes)
	}
	ix := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if name != "" {
		ix.Content = append(ix.Content, scalar("name"), scalar(name))
	}
	vals := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
	for _, f := range fields {
		vals.Content = append(vals.Content, scalar(f))
	}
	ix.Content = append(ix.Content, scalar("fields"), vals)
	if unique {
		ix.Content = append(ix.Content, scalar("unique"), &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	}
	if method != "" && method != "btree" {
		ix.Content = append(ix.Content, scalar("method"), scalar(method))
	}
	if len(include) > 0 {
		vals := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, f := range include {
			vals.Content = append(vals.Content, scalar(f))
		}
		ix.Content = append(ix.Content, scalar("include"), vals)
	}
	if where != "" {
		ix.Content = append(ix.Content, scalar("where"), scalar(where))
	}
	indexes.Content = append(indexes.Content, ix)
	return encodeRoot(root)
}

func domainAddView(s *session, args []string, dryRun bool) (string, any, error) {
	usage := fmt.Errorf("usage: domain add-view <app> <name> (--path <file> | <kind> --model <model>) [<<EOF … EOF]")
	var pos []string
	model, path := "", ""
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--model", "--path":
			if i+1 >= len(args) {
				return "", nil, usage
			}
			i++
			if a == "--model" {
				model = args[i]
			} else {
				path = args[i]
			}
		default:
			if strings.HasPrefix(a, "--") {
				return "", nil, fmt.Errorf("unknown flag %s", a)
			}
			pos = append(pos, a)
		}
	}
	body := ""
	if path != "" {
		if len(pos) == 3 {
			body = pos[2]
			pos = pos[:2]
		}
		if len(pos) != 2 {
			return "", nil, usage
		}
		d, err := fetchDomain(s, pos[0])
		if err != nil {
			return "", nil, err
		}
		doc, err := addViewToDoc(fmt.Sprint(d["spec"]), pos[1], "", "", path, body)
		if err != nil {
			return "", nil, err
		}
		return applyDomain(s, doc, dryRun, false, migrateOpts{})
	}
	if len(pos) == 4 {
		body = pos[3]
		pos = pos[:3]
	}
	if len(pos) != 3 || model == "" {
		return "", nil, usage
	}
	d, err := fetchDomain(s, pos[0])
	if err != nil {
		return "", nil, err
	}
	doc, err := addViewToDoc(fmt.Sprint(d["spec"]), pos[1], pos[2], model, "", body)
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, false, migrateOpts{})
}

func addViewToDoc(doc, name, kind, model, path, body string) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return "", fmt.Errorf("parse domain: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("domain document is not a mapping")
	}
	top := root.Content[0]
	views := mapValue(top, "views")
	if views == nil {
		views = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		top.Content = append(top.Content, scalar("views"), views)
	}
	for _, existing := range views.Content {
		if n := mapValue(existing, "name"); n != nil && n.Value == name {
			return "", fmt.Errorf("view %q already exists", name)
		}
	}
	view := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if strings.TrimSpace(body) != "" && path == "" {
		var bodyNode yaml.Node
		if err := yaml.Unmarshal([]byte(body), &bodyNode); err != nil {
			return "", fmt.Errorf("view body: %w", err)
		}
		if len(bodyNode.Content) == 0 || bodyNode.Content[0].Kind != yaml.MappingNode {
			return "", fmt.Errorf("view body must be a mapping")
		}
		view = bodyNode.Content[0]
	}
	setMap(view, "name", scalar(name))
	if path != "" {
		setMap(view, "path", scalar(path))
	} else if mapValue(view, "components") == nil {
		comp := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMap(comp, "id", scalar("main"))
		setMap(comp, "kind", scalar(kind))
		setMap(comp, "model", scalar(model))
		for _, key := range []string{"columns", "groups", "filters", "groupBy", "defaultView", "views"} {
			if n := mapValue(view, key); n != nil {
				setMap(comp, key, n)
				deleteMapKey(view, key)
			}
		}
		deleteMapKey(view, "kind")
		deleteMapKey(view, "model")
		setMap(view, "layout", scalar("column"))
		setMap(view, "components", &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: []*yaml.Node{comp}})
	}
	views.Content = append(views.Content, view)
	return encodeRoot(&root)
}

func modelNode(doc, model string) (*yaml.Node, *yaml.Node, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return nil, nil, fmt.Errorf("parse domain: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("domain document is not a mapping")
	}
	models := mapValue(root.Content[0], "models")
	if models == nil || models.Kind != yaml.SequenceNode {
		return nil, nil, fmt.Errorf("domain has no models list")
	}
	for _, m := range models.Content {
		if n := mapValue(m, "name"); n != nil && n.Value == model {
			return &root, m, nil
		}
	}
	return nil, nil, fmt.Errorf("model %q not found", model)
}

func setMap(m *yaml.Node, key string, val *yaml.Node) {
	if existing := mapValue(m, key); existing != nil {
		*existing = *val
		return
	}
	m.Content = append(m.Content, scalar(key), val)
}

func deleteMapKey(m *yaml.Node, key string) {
	if m == nil || m.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(m.Content)-1; i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}

func encodeRoot(root *yaml.Node) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
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

func cloneMapping(n *yaml.Node) *yaml.Node {
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if n == nil || n.Kind != yaml.MappingNode {
		return out
	}
	out.Content = make([]*yaml.Node, len(n.Content))
	copy(out.Content, n.Content)
	return out
}

func setFieldInDoc(doc, model, oldName string, f newField) (string, error) {
	root, target, err := modelNode(doc, model)
	if err != nil {
		return "", err
	}
	fields := mapValue(target, "fields")
	if fields == nil || fields.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("model %s has no fields", model)
	}
	idx := -1
	for i, existing := range fields.Content {
		if n := mapValue(existing, "name"); n != nil && n.Value == oldName {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("model %s has no field %q", model, oldName)
	}
	if f.Name != oldName {
		for _, existing := range fields.Content {
			if n := mapValue(existing, "name"); n != nil && n.Value == f.Name {
				return "", fmt.Errorf("model %s already has field %q", model, f.Name)
			}
		}
	}
	var field *yaml.Node
	if f.Name != oldName {
		// Rename: keep other keys (required, indexed, …), only change name/type + explicit flags.
		field = cloneMapping(fields.Content[idx])
		setMap(field, "name", scalar(f.Name))
		setMap(field, "type", scalar(f.Type))
	} else {
		field = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		field.Content = append(field.Content, scalar("name"), scalar(f.Name), scalar("type"), scalar(f.Type))
	}
	for _, flag := range []struct {
		key string
		on  bool
	}{{"required", f.Required}, {"indexed", f.Indexed}, {"unique", f.Unique}, {"readonly", f.Readonly}} {
		if flag.on {
			setMap(field, flag.key, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
		}
	}
	if len(f.Values) > 0 {
		vals := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, v := range f.Values {
			vals.Content = append(vals.Content, scalar(v))
		}
		setMap(field, "values", vals)
	}
	if f.Relation != "" {
		setMap(field, "relation", scalar(f.Relation))
	}
	if f.OnDelete != "" {
		setMap(field, "onDelete", scalar(f.OnDelete))
	}
	appendComputeKeys(field, f)
	fields.Content[idx] = field
	if f.Name != oldName {
		rewriteFieldRefs(root.Content[0], model, oldName, f.Name)
		rewriteModelSearchAndIndexes(target, oldName, f.Name)
	}
	return encodeRoot(root)
}

// rewriteFieldRefs updates view column/groupBy field names for views bound to model.
func rewriteFieldRefs(doc *yaml.Node, model, oldName, newName string) {
	views := mapValue(doc, "views")
	if views == nil || views.Kind != yaml.SequenceNode {
		return
	}
	for _, v := range views.Content {
		rewriteBoundFieldRefs(v, model, oldName, newName)
		if comps := mapValue(v, "components"); comps != nil && comps.Kind == yaml.SequenceNode {
			for _, c := range comps.Content {
				rewriteBoundFieldRefs(c, model, oldName, newName)
			}
		}
	}
}

func rewriteBoundFieldRefs(node *yaml.Node, model, oldName, newName string) {
	if n := mapValue(node, "model"); n == nil || n.Value != model {
		return
	}
	rewriteStringSeq(mapValue(node, "groupBy"), oldName, newName)
	if cols := mapValue(node, "columns"); cols != nil && cols.Kind == yaml.SequenceNode {
		for _, c := range cols.Content {
			if fn := mapValue(c, "field"); fn != nil && fn.Value == oldName {
				fn.Value = newName
			}
		}
	}
	if groups := mapValue(node, "groups"); groups != nil && groups.Kind == yaml.SequenceNode {
		for _, g := range groups.Content {
			if fs := mapValue(g, "fields"); fs != nil && fs.Kind == yaml.SequenceNode {
				for _, c := range fs.Content {
					if fn := mapValue(c, "field"); fn != nil && fn.Value == oldName {
						fn.Value = newName
					}
				}
			}
		}
	}
}

func rewriteModelSearchAndIndexes(modelNode *yaml.Node, oldName, newName string) {
	if search := mapValue(modelNode, "search"); search != nil {
		rewriteStringSeq(mapValue(search, "fields"), oldName, newName)
	}
	indexes := mapValue(modelNode, "indexes")
	if indexes == nil || indexes.Kind != yaml.SequenceNode {
		return
	}
	for _, ix := range indexes.Content {
		rewriteStringSeq(mapValue(ix, "fields"), oldName, newName)
	}
}

func rewriteStringSeq(seq *yaml.Node, oldName, newName string) {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return
	}
	for _, n := range seq.Content {
		if n.Kind == yaml.ScalarNode && n.Value == oldName {
			n.Value = newName
		}
	}
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func pluralizeModel(s string) string {
	if s == "" {
		return s
	}
	if strings.HasSuffix(s, "y") && len(s) > 1 {
		prev := s[len(s)-2]
		if prev != 'a' && prev != 'e' && prev != 'i' && prev != 'o' && prev != 'u' {
			return s[:len(s)-1] + "ies"
		}
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "z") ||
		strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}

func fieldColName(field string) string {
	switch field {
	case "orgId":
		return "org_id"
	case "authorId":
		return "author_id"
	case "createdAt":
		return "created_at"
	case "updatedAt":
		return "updated_at"
	default:
		var b strings.Builder
		for i, r := range field {
			if r >= 'A' && r <= 'Z' {
				if i > 0 {
					b.WriteByte('_')
				}
				b.WriteRune(r - 'A' + 'a')
			} else {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
}

func removeFieldFromDoc(doc, model, field string) (string, error) {
	root, target, err := modelNode(doc, model)
	if err != nil {
		return "", err
	}
	fields := mapValue(target, "fields")
	if fields == nil || fields.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("model %s has no fields", model)
	}
	kept := make([]*yaml.Node, 0, len(fields.Content))
	found := false
	for _, existing := range fields.Content {
		if n := mapValue(existing, "name"); n != nil && n.Value == field {
			found = true
			continue
		}
		kept = append(kept, existing)
	}
	if !found {
		return "", fmt.Errorf("model %s has no field %q", model, field)
	}
	fields.Content = kept
	return encodeRoot(root)
}

func domainSetIndex(s *session, args []string, dryRun bool) (string, any, error) {
	usage := fmt.Errorf("usage: domain set-index <app>.<model> <name> <field>[,field…] [--unique] [--method btree|hash|gin] [--include a,b] [--where \"col = 'v'\"] [--dry-run]")
	var pos []string
	method, where := "btree", ""
	var include []string
	unique := false
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--unique":
			unique = true
		case "--method", "--include", "--where":
			if i+1 >= len(args) {
				return "", nil, usage
			}
			i++
			switch a {
			case "--method":
				method = args[i]
			case "--include":
				include = splitList(args[i])
			default:
				where = args[i]
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
	if !ok {
		return "", nil, usage
	}
	d, err := fetchDomain(s, app)
	if err != nil {
		return "", nil, err
	}
	doc, err := setIndexInDoc(fmt.Sprint(d["spec"]), model, pos[1], method, unique, splitList(pos[2]), include, where)
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, false, migrateOpts{})
}

func domainRemoveIndex(s *session, args []string, dryRun, force bool) (string, any, error) {
	usage := fmt.Errorf("usage: domain remove-index <app>.<model> <name-or-fields> [--force] [--dry-run]")
	if len(args) != 2 {
		return "", nil, usage
	}
	app, model, ok := strings.Cut(args[0], ".")
	if !ok {
		return "", nil, usage
	}
	d, err := fetchDomain(s, app)
	if err != nil {
		return "", nil, err
	}
	doc, err := removeIndexFromDoc(fmt.Sprint(d["spec"]), model, args[1])
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, force, migrateOpts{})
}

func setIndexInDoc(doc, model, name, method string, unique bool, fields, include []string, where string) (string, error) {
	root, target, err := modelNode(doc, model)
	if err != nil {
		return "", err
	}
	indexes := mapValue(target, "indexes")
	if indexes == nil {
		return "", fmt.Errorf("model %s has no indexes", model)
	}
	idx := -1
	for i, ix := range indexes.Content {
		if n := mapValue(ix, "name"); n != nil && n.Value == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", fmt.Errorf("index %q not found on model %s", name, model)
	}
	ix := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	ix.Content = append(ix.Content, scalar("name"), scalar(name))
	vals := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
	for _, f := range fields {
		vals.Content = append(vals.Content, scalar(f))
	}
	ix.Content = append(ix.Content, scalar("fields"), vals)
	if unique {
		ix.Content = append(ix.Content, scalar("unique"), &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	}
	if method != "" && method != "btree" {
		ix.Content = append(ix.Content, scalar("method"), scalar(method))
	}
	if len(include) > 0 {
		inc := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, f := range include {
			inc.Content = append(inc.Content, scalar(f))
		}
		ix.Content = append(ix.Content, scalar("include"), inc)
	}
	if where != "" {
		ix.Content = append(ix.Content, scalar("where"), scalar(where))
	}
	indexes.Content[idx] = ix
	return encodeRoot(root)
}

func removeIndexFromDoc(doc, model, key string) (string, error) {
	root, target, err := modelNode(doc, model)
	if err != nil {
		return "", err
	}
	indexes := mapValue(target, "indexes")
	if indexes == nil {
		return "", fmt.Errorf("model %s has no indexes", model)
	}
	kept := make([]*yaml.Node, 0, len(indexes.Content))
	found := false
	for _, ix := range indexes.Content {
		match := false
		if n := mapValue(ix, "name"); n != nil && n.Value == key {
			match = true
		} else if f := mapValue(ix, "fields"); f != nil {
			var parts []string
			for _, c := range f.Content {
				parts = append(parts, c.Value)
			}
			if strings.Join(parts, ",") == key {
				match = true
			}
		}
		if match {
			found = true
			continue
		}
		kept = append(kept, ix)
	}
	if !found {
		return "", fmt.Errorf("index %q not found on model %s", key, model)
	}
	indexes.Content = kept
	return encodeRoot(root)
}

func domainSetView(s *session, args []string, dryRun, ted bool) (string, any, error) {
	usage := fmt.Errorf("usage: domain set-view <app> <name> [-ted | <<EOF … EOF]")
	if len(args) < 2 {
		return "", nil, usage
	}
	body := ""
	pos := args
	if len(args) >= 3 {
		body = args[len(args)-1]
		pos = args[:len(args)-1]
	}
	if len(pos) != 2 {
		return "", nil, usage
	}
	if ted && strings.TrimSpace(body) == "" {
		d, err := fetchDomain(s, pos[0])
		if err != nil {
			return "", nil, err
		}
		seed, err := extractNamedYAML(fmt.Sprint(d["spec"]), "views", "name", pos[1])
		if err != nil {
			seed = "layout: column\ncomponents:\n  - id: main\n    kind: collection\n    model:\n"
		}
		return tedSoft(fmt.Sprintf("domain set-view %s %s", pos[0], pos[1]), seed)
	}
	if strings.TrimSpace(body) == "" {
		return "", nil, usage
	}
	d, err := fetchDomain(s, pos[0])
	if err != nil {
		return "", nil, err
	}
	doc, err := setViewInDoc(fmt.Sprint(d["spec"]), pos[1], body)
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, false, migrateOpts{})
}

func setViewInDoc(doc, name, body string) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return "", fmt.Errorf("parse domain: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("domain document is not a mapping")
	}
	views := mapValue(root.Content[0], "views")
	if views == nil || views.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("domain has no views")
	}
	var bodyNode yaml.Node
	if err := yaml.Unmarshal([]byte(body), &bodyNode); err != nil {
		return "", fmt.Errorf("view body: %w", err)
	}
	if len(bodyNode.Content) == 0 || bodyNode.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("view body must be a mapping")
	}
	view := bodyNode.Content[0]
	setMap(view, "name", scalar(name))
	for i, existing := range views.Content {
		if n := mapValue(existing, "name"); n != nil && n.Value == name {
			views.Content[i] = view
			return encodeRoot(&root)
		}
	}
	views.Content = append(views.Content, view)
	return encodeRoot(&root)
}

func domainRemoveNamed(s *session, args []string, dryRun, force bool, listKey, kind string) (string, any, error) {
	usage := fmt.Errorf("usage: domain remove-%s <app> <name> [--force] [--dry-run]", kind)
	if len(args) != 2 {
		return "", nil, usage
	}
	d, err := fetchDomain(s, args[0])
	if err != nil {
		return "", nil, err
	}
	nameKey := "name"
	if listKey == "menus" {
		nameKey = "id"
	}
	doc, err := removeFromList(fmt.Sprint(d["spec"]), listKey, nameKey, args[1])
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, force, migrateOpts{})
}

func removeFromList(doc, listKey, nameKey, name string) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return "", fmt.Errorf("parse domain: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("domain document is not a mapping")
	}
	list := mapValue(root.Content[0], listKey)
	if list == nil || list.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("domain has no %s", listKey)
	}
	kept := make([]*yaml.Node, 0, len(list.Content))
	found := false
	for _, item := range list.Content {
		if n := mapValue(item, nameKey); n != nil && n.Value == name {
			found = true
			continue
		}
		kept = append(kept, item)
	}
	if !found {
		return "", fmt.Errorf("%s %q not found", listKey, name)
	}
	list.Content = kept
	return encodeRoot(&root)
}

func domainAddMenu(s *session, args []string, dryRun bool) (string, any, error) {
	return domainUpsertMenu(s, args, dryRun, false, false)
}

func domainSetMenu(s *session, args []string, dryRun, ted bool) (string, any, error) {
	return domainUpsertMenu(s, args, dryRun, true, ted)
}

func domainUpsertMenu(s *session, args []string, dryRun, replace, ted bool) (string, any, error) {
	usage := fmt.Errorf("usage: domain %s-menu <app> <id> [--label x] [--view V] [--model m] [--route r] [-ted] [--dry-run] | <<EOF body EOF",
		map[bool]string{true: "set", false: "add"}[replace])
	var pos []string
	label, view, model, route := "", "", "", ""
	body := ""
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--label", "--view", "--model", "--route":
			if i+1 >= len(args) {
				return "", nil, usage
			}
			i++
			switch a {
			case "--label":
				label = args[i]
			case "--view":
				view = args[i]
			case "--model":
				model = args[i]
			default:
				route = args[i]
			}
		default:
			if strings.HasPrefix(a, "--") {
				return "", nil, fmt.Errorf("unknown flag %s", a)
			}
			pos = append(pos, a)
		}
	}
	if len(pos) == 3 {
		body = pos[2]
		pos = pos[:2]
	}
	if len(pos) != 2 {
		return "", nil, usage
	}
	if ted && replace && strings.TrimSpace(body) == "" {
		d, err := fetchDomain(s, pos[0])
		if err != nil {
			return "", nil, err
		}
		seed, err := extractNamedYAML(fmt.Sprint(d["spec"]), "menus", "id", pos[1])
		if err != nil {
			seed = "label: \n"
		}
		return tedSoft(fmt.Sprintf("domain set-menu %s %s", pos[0], pos[1]), seed)
	}
	d, err := fetchDomain(s, pos[0])
	if err != nil {
		return "", nil, err
	}
	doc, err := upsertMenuInDoc(fmt.Sprint(d["spec"]), pos[1], label, view, model, route, body, replace)
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, false, migrateOpts{})
}

func upsertMenuInDoc(doc, id, label, view, model, route, body string, replace bool) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return "", err
	}
	top := root.Content[0]
	menus := mapValue(top, "menus")
	if menus == nil {
		menus = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		top.Content = append(top.Content, scalar("menus"), menus)
	}
	menu := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if strings.TrimSpace(body) != "" {
		var bodyNode yaml.Node
		if err := yaml.Unmarshal([]byte(body), &bodyNode); err != nil {
			return "", fmt.Errorf("menu body: %w", err)
		}
		menu = bodyNode.Content[0]
	}
	setMap(menu, "id", scalar(id))
	if label != "" {
		setMap(menu, "label", scalar(label))
	}
	if view != "" {
		setMap(menu, "view", scalar(view))
	}
	if model != "" {
		setMap(menu, "model", scalar(model))
	}
	if route != "" {
		setMap(menu, "route", scalar(route))
	}
	for i, existing := range menus.Content {
		if n := mapValue(existing, "id"); n != nil && n.Value == id {
			if !replace {
				return "", fmt.Errorf("menu %q already exists", id)
			}
			menus.Content[i] = menu
			return encodeRoot(&root)
		}
	}
	if replace {
		return "", fmt.Errorf("menu %q not found", id)
	}
	menus.Content = append(menus.Content, menu)
	return encodeRoot(&root)
}

func domainAddCQRS(s *session, args []string, dryRun bool, listKey, kind string, replace, ted bool) (string, any, error) {
	usage := fmt.Errorf("usage: domain %s-%s <app> <name> [-ted | <<EOF … EOF]", map[bool]string{true: "set", false: "add"}[replace], kind)
	if len(args) < 2 {
		return "", nil, usage
	}
	body := ""
	pos := args
	if len(args) >= 3 {
		body = args[len(args)-1]
		pos = args[:len(args)-1]
	}
	if len(pos) != 2 {
		return "", nil, usage
	}
	if ted && replace && strings.TrimSpace(body) == "" {
		d, err := fetchDomain(s, pos[0])
		if err != nil {
			return "", nil, err
		}
		seed, err := extractNamedYAML(fmt.Sprint(d["spec"]), listKey, "name", pos[1])
		if err != nil {
			seed = ""
		}
		return tedSoft(fmt.Sprintf("domain set-%s %s %s", kind, pos[0], pos[1]), seed)
	}
	if strings.TrimSpace(body) == "" {
		return "", nil, usage
	}
	d, err := fetchDomain(s, pos[0])
	if err != nil {
		return "", nil, err
	}
	doc, err := upsertNamedInList(fmt.Sprint(d["spec"]), listKey, pos[1], body, replace)
	if err != nil {
		return "", nil, err
	}
	return applyDomain(s, doc, dryRun, false, migrateOpts{})
}

func upsertNamedInList(doc, listKey, name, body string, replace bool) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return "", err
	}
	top := root.Content[0]
	list := mapValue(top, listKey)
	if list == nil {
		list = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		top.Content = append(top.Content, scalar(listKey), list)
	}
	var bodyNode yaml.Node
	if err := yaml.Unmarshal([]byte(body), &bodyNode); err != nil {
		return "", err
	}
	item := bodyNode.Content[0]
	setMap(item, "name", scalar(name))
	for i, existing := range list.Content {
		if n := mapValue(existing, "name"); n != nil && n.Value == name {
			if !replace {
				return "", fmt.Errorf("%s %q already exists", listKey, name)
			}
			list.Content[i] = item
			return encodeRoot(&root)
		}
	}
	if replace {
		return "", fmt.Errorf("%s %q not found", listKey, name)
	}
	list.Content = append(list.Content, item)
	return encodeRoot(&root)
}
