package appspec

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// inferDepends fills depends from a compute expression or the first related hop.
func inferDepends(f FieldSpec) []string {
	if f.HookCompute() {
		return nil
	}
	if rel := strings.TrimSpace(f.Related); rel != "" {
		head, _, _ := strings.Cut(rel, ".")
		if head == "" {
			return nil
		}
		return []string{head}
	}
	expr := strings.TrimSpace(f.Compute)
	if expr == "" || f.HookCompute() {
		return nil
	}
	ids, err := ExprIdents(expr)
	if err != nil {
		return nil
	}
	return ids
}

func validateComputes(spec AppSpec, m ModelSpec, fields map[string]FieldSpec) error {
	for _, f := range m.Fields {
		if strings.TrimSpace(f.Compute) != "" && strings.TrimSpace(f.Related) != "" {
			return fmt.Errorf("model %q field %q sets both compute and related", m.Name, f.Name)
		}
		if f.Store != nil && !f.Computed() {
			return fmt.Errorf("model %q field %q: store is only for compute or related", m.Name, f.Name)
		}
		if f.IsRelation() && f.Computed() {
			return fmt.Errorf("model %q field %q: relation fields are not computed", m.Name, f.Name)
		}
		for _, dep := range f.Depends {
			if dep == f.Name {
				return fmt.Errorf("model %q field %q depends on itself", m.Name, f.Name)
			}
			if _, ok := fields[dep]; !ok {
				return fmt.Errorf("model %q field %q depends on unknown field %q", m.Name, f.Name, dep)
			}
		}
		switch {
		case f.HookCompute():
		case strings.TrimSpace(f.Compute) != "":
			ids, err := ExprIdents(f.Compute)
			if err != nil {
				return fmt.Errorf("model %q field %q: %w", m.Name, f.Name, err)
			}
			for _, id := range ids {
				if _, ok := fields[id]; !ok {
					return fmt.Errorf("model %q field %q compute references unknown field %q", m.Name, f.Name, id)
				}
			}
		case strings.TrimSpace(f.Related) != "":
			if err := validateRelated(spec, m.Name, f, fields); err != nil {
				return err
			}
		}
	}
	if _, err := OrderedComputes(m.Name, m.Fields); err != nil {
		return err
	}
	return nil
}

func validateRelated(spec AppSpec, model string, f FieldSpec, fields map[string]FieldSpec) error {
	parts := strings.Split(strings.TrimSpace(f.Related), ".")
	if len(parts) < 2 {
		return fmt.Errorf("model %q field %q related %q needs a path like topicId.name", model, f.Name, f.Related)
	}
	for _, part := range parts {
		if !fieldNameRe.MatchString(part) {
			return fmt.Errorf("model %q field %q has invalid related segment %q", model, f.Name, part)
		}
	}
	models := map[string]ModelSpec{}
	for _, m := range spec.Models {
		models[m.Name] = m
	}
	cursorName := model
	cursorFields := fields
	for i := 0; i < len(parts)-1; i++ {
		hop, ok := cursorFields[parts[i]]
		if !ok {
			return fmt.Errorf("model %q field %q related path has no field %s.%s", model, f.Name, cursorName, parts[i])
		}
		if hop.CanonicalType() != TypeMany2One {
			return fmt.Errorf("model %q field %q related hop %s.%s must be many2one", model, f.Name, cursorName, hop.Name)
		}
		ref, ok := hop.ResolveRelation(spec.Name)
		if !ok {
			return fmt.Errorf("model %q field %q related hop %s has no relation target", model, f.Name, hop.Name)
		}
		if ref.App != "" && ref.App != spec.Name {
			return nil
		}
		target, ok := models[ref.Model]
		if !ok {
			return nil
		}
		cursorName = target.Name
		cursorFields = fieldMap(target)
	}
	last := parts[len(parts)-1]
	target, ok := cursorFields[last]
	if !ok {
		return fmt.Errorf("model %q field %q related path has no field %s.%s", model, f.Name, cursorName, last)
	}
	if !target.Column() {
		return fmt.Errorf("model %q field %q related path must end on a column", model, f.Name)
	}
	if target.CanonicalType() != f.CanonicalType() {
		return fmt.Errorf("model %q field %q type %s does not match %s.%s (%s)", model, f.Name, f.CanonicalType(), cursorName, last, target.CanonicalType())
	}
	return nil
}

// validateInverses checks one2many inverses when the target model is in this spec.
func validateInverses(s AppSpec) error {
	models := map[string]ModelSpec{}
	for _, m := range s.Models {
		models[m.Name] = m
	}
	for _, m := range s.Models {
		for _, f := range m.Fields {
			if f.CanonicalType() != TypeOne2Many {
				continue
			}
			ref, ok := f.ResolveRelation(s.Name)
			if !ok || (ref.App != "" && ref.App != s.Name) {
				continue
			}
			target, ok := models[ref.Model]
			if !ok {
				continue
			}
			inv, ok := fieldMap(target)[f.Inverse]
			if !ok {
				return fmt.Errorf("model %q field %q inverse %q is not a field on %s", m.Name, f.Name, f.Inverse, target.Name)
			}
			if inv.CanonicalType() != TypeMany2One {
				return fmt.Errorf("model %q field %q inverse %s.%s must be many2one", m.Name, f.Name, target.Name, inv.Name)
			}
			back, ok := inv.ResolveRelation(s.Name)
			if !ok || back.Model != m.Name || (back.App != "" && back.App != s.Name) {
				return fmt.Errorf("model %q field %q inverse %s.%s must relate back to %s", m.Name, f.Name, target.Name, inv.Name, m.Name)
			}
		}
	}
	return nil
}

func fieldMap(m ModelSpec) map[string]FieldSpec {
	out := map[string]FieldSpec{}
	for _, f := range m.Fields {
		out[f.Name] = f
	}
	return out
}

// OrderedComputes returns computed fields so each dependency is produced first.
func OrderedComputes(model string, fields []FieldSpec) ([]FieldSpec, error) {
	byName := map[string]FieldSpec{}
	var nodes []string
	for _, f := range fields {
		if !f.Computed() {
			continue
		}
		byName[f.Name] = f
		nodes = append(nodes, f.Name)
	}
	next := map[string][]string{}
	indeg := map[string]int{}
	for _, name := range nodes {
		indeg[name] = 0
	}
	for _, name := range nodes {
		for _, dep := range byName[name].Depends {
			if _, ok := byName[dep]; !ok {
				continue
			}
			next[dep] = append(next[dep], name)
			indeg[name]++
		}
	}
	var q []string
	for _, name := range nodes {
		if indeg[name] == 0 {
			q = append(q, name)
		}
	}
	var out []FieldSpec
	for len(q) > 0 {
		n := q[0]
		q = q[1:]
		out = append(out, byName[n])
		for _, m := range next[n] {
			indeg[m]--
			if indeg[m] == 0 {
				q = append(q, m)
			}
		}
	}
	if len(out) != len(nodes) {
		return nil, fmt.Errorf("model %q computed fields have a cycle", model)
	}
	return out, nil
}

// ExprIdents returns field names referenced by a compute expression.
func ExprIdents(expr string) ([]string, error) {
	_, ids, err := evalExpr(expr, nil)
	return ids, err
}

// EvalCompute evaluates a field expression against the current record values.
// Identifiers are field names. Operators: + - * / == != && ||. Functions:
// len, lower, upper, trim, coalesce.
func EvalCompute(expr string, fields map[string]any) (any, error) {
	v, _, err := evalExpr(expr, fields)
	if err != nil {
		return nil, err
	}
	return v.any(), nil
}

type valKind int

const (
	valNull valKind = iota
	valNum
	valStr
	valBool
)

type val struct {
	k valKind
	n float64
	s string
	b bool
}

func (v val) any() any {
	switch v.k {
	case valNum:
		return v.n
	case valStr:
		return v.s
	case valBool:
		return v.b
	default:
		return nil
	}
}

func (v val) display() string {
	switch v.k {
	case valNull:
		return ""
	case valNum:
		return strconv.FormatFloat(v.n, 'f', -1, 64)
	case valBool:
		if v.b {
			return "true"
		}
		return "false"
	default:
		return v.s
	}
}

func (v val) truthy() bool {
	switch v.k {
	case valBool:
		return v.b
	case valNum:
		return v.n != 0
	case valStr:
		return strings.TrimSpace(v.s) != ""
	default:
		return false
	}
}

func valFrom(v any) val {
	switch t := v.(type) {
	case nil:
		return val{}
	case val:
		return t
	case bool:
		return val{k: valBool, b: t}
	case float64:
		return val{k: valNum, n: t}
	case float32:
		return val{k: valNum, n: float64(t)}
	case int:
		return val{k: valNum, n: float64(t)}
	case int32:
		return val{k: valNum, n: float64(t)}
	case int64:
		return val{k: valNum, n: float64(t)}
	case string:
		return val{k: valStr, s: t}
	default:
		return val{k: valStr, s: strings.TrimSpace(fmt.Sprint(t))}
	}
}

type tokKind int

const (
	tokEOF tokKind = iota
	tokIdent
	tokString
	tokNumber
	tokOp
	tokLParen
	tokRParen
	tokComma
)

type token struct {
	kind tokKind
	text string
	num  float64
}

type parser struct {
	src    string
	toks   []token
	i      int
	idents []string
	seen   map[string]struct{}
}

func evalExpr(expr string, fields map[string]any) (val, []string, error) {
	p := &parser{src: expr, seen: map[string]struct{}{}}
	if err := p.lex(); err != nil {
		return val{}, nil, err
	}
	v, err := p.parseOr(fields)
	if err != nil {
		return val{}, nil, err
	}
	if p.peek().kind != tokEOF {
		return val{}, nil, fmt.Errorf("compute: unexpected %q", p.peek().text)
	}
	return v, p.idents, nil
}

func (p *parser) noteIdent(name string) {
	if _, ok := p.seen[name]; ok {
		return
	}
	p.seen[name] = struct{}{}
	p.idents = append(p.idents, name)
}

func (p *parser) lex() error {
	s := p.src
	i := 0
	for i < len(s) {
		r, w := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			i += w
			continue
		}
		rest := s[i:]
		switch {
		case strings.HasPrefix(rest, "||"), strings.HasPrefix(rest, "&&"), strings.HasPrefix(rest, "=="), strings.HasPrefix(rest, "!="):
			p.toks = append(p.toks, token{kind: tokOp, text: rest[:2]})
			i += 2
			continue
		}
		switch s[i] {
		case '(', ')', ',':
			k := tokLParen
			if s[i] == ')' {
				k = tokRParen
			} else if s[i] == ',' {
				k = tokComma
			}
			p.toks = append(p.toks, token{kind: k, text: s[i : i+1]})
			i++
		case '+', '-', '*', '/':
			p.toks = append(p.toks, token{kind: tokOp, text: s[i : i+1]})
			i++
		case '"', '\'':
			text, n, err := scanString(s[i:])
			if err != nil {
				return err
			}
			p.toks = append(p.toks, token{kind: tokString, text: text})
			i += n
		default:
			if s[i] >= '0' && s[i] <= '9' {
				j := i + 1
				for j < len(s) && (s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
					j++
				}
				n, err := strconv.ParseFloat(s[i:j], 64)
				if err != nil {
					return fmt.Errorf("compute: bad number %q", s[i:j])
				}
				p.toks = append(p.toks, token{kind: tokNumber, text: s[i:j], num: n})
				i = j
				continue
			}
			if isIdentStart(r) {
				j := i + w
				for j < len(s) {
					rr, ww := utf8.DecodeRuneInString(s[j:])
					if !isIdentCont(rr) {
						break
					}
					j += ww
				}
				p.toks = append(p.toks, token{kind: tokIdent, text: s[i:j]})
				i = j
				continue
			}
			return fmt.Errorf("compute: unexpected %q", string(r))
		}
	}
	p.toks = append(p.toks, token{kind: tokEOF})
	return nil
}

func scanString(s string) (string, int, error) {
	quote := s[0]
	var b strings.Builder
	i := 1
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				b.WriteByte('\n')
			case '\\', '"', '\'':
				b.WriteByte(s[i+1])
			default:
				b.WriteByte(s[i+1])
			}
			i += 2
			continue
		}
		if s[i] == quote {
			return b.String(), i + 1, nil
		}
		b.WriteByte(s[i])
		i++
	}
	return "", 0, fmt.Errorf("compute: unterminated string")
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentCont(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func (p *parser) peek() token {
	if p.i >= len(p.toks) {
		return token{kind: tokEOF}
	}
	return p.toks[p.i]
}

func (p *parser) next() token {
	t := p.peek()
	if p.i < len(p.toks) {
		p.i++
	}
	return t
}

func (p *parser) parseOr(fields map[string]any) (val, error) {
	left, err := p.parseAnd(fields)
	if err != nil {
		return val{}, err
	}
	for p.peek().kind == tokOp && p.peek().text == "||" {
		p.next()
		right, err := p.parseAnd(fields)
		if err != nil {
			return val{}, err
		}
		left = val{k: valBool, b: left.truthy() || right.truthy()}
	}
	return left, nil
}

func (p *parser) parseAnd(fields map[string]any) (val, error) {
	left, err := p.parseCmp(fields)
	if err != nil {
		return val{}, err
	}
	for p.peek().kind == tokOp && p.peek().text == "&&" {
		p.next()
		right, err := p.parseCmp(fields)
		if err != nil {
			return val{}, err
		}
		left = val{k: valBool, b: left.truthy() && right.truthy()}
	}
	return left, nil
}

func (p *parser) parseCmp(fields map[string]any) (val, error) {
	left, err := p.parseAdd(fields)
	if err != nil {
		return val{}, err
	}
	for p.peek().kind == tokOp && (p.peek().text == "==" || p.peek().text == "!=") {
		op := p.next().text
		right, err := p.parseAdd(fields)
		if err != nil {
			return val{}, err
		}
		eq := cmpVal(left, right)
		if op == "!=" {
			eq = !eq
		}
		left = val{k: valBool, b: eq}
	}
	return left, nil
}

func cmpVal(a, b val) bool {
	if a.k == valNull || b.k == valNull {
		return a.k == valNull && b.k == valNull
	}
	if a.k == valNum && b.k == valNum {
		return a.n == b.n
	}
	if a.k == valBool && b.k == valBool {
		return a.b == b.b
	}
	if a.k == valStr && b.k == valStr {
		return a.s == b.s
	}
	return a.display() == b.display()
}

func (p *parser) parseAdd(fields map[string]any) (val, error) {
	left, err := p.parseMul(fields)
	if err != nil {
		return val{}, err
	}
	for p.peek().kind == tokOp && (p.peek().text == "+" || p.peek().text == "-") {
		op := p.next().text
		right, err := p.parseMul(fields)
		if err != nil {
			return val{}, err
		}
		left, err = arith(op, left, right)
		if err != nil {
			return val{}, err
		}
	}
	return left, nil
}

func (p *parser) parseMul(fields map[string]any) (val, error) {
	left, err := p.parseUnary(fields)
	if err != nil {
		return val{}, err
	}
	for p.peek().kind == tokOp && (p.peek().text == "*" || p.peek().text == "/") {
		op := p.next().text
		right, err := p.parseUnary(fields)
		if err != nil {
			return val{}, err
		}
		left, err = arith(op, left, right)
		if err != nil {
			return val{}, err
		}
	}
	return left, nil
}

func arith(op string, left, right val) (val, error) {
	if op == "+" && (left.k == valStr || right.k == valStr) {
		return val{k: valStr, s: left.display() + right.display()}, nil
	}
	ln, rn := numOrZero(left), numOrZero(right)
	if left.k == valStr || right.k == valStr {
		return val{}, fmt.Errorf("compute: %s is not a number", op)
	}
	switch op {
	case "+":
		return val{k: valNum, n: ln + rn}, nil
	case "-":
		return val{k: valNum, n: ln - rn}, nil
	case "*":
		return val{k: valNum, n: ln * rn}, nil
	case "/":
		if rn == 0 {
			return val{}, fmt.Errorf("compute: division by zero")
		}
		return val{k: valNum, n: ln / rn}, nil
	default:
		return val{}, fmt.Errorf("compute: unknown operator %s", op)
	}
}

func numOrZero(v val) float64 {
	if v.k == valNum {
		return v.n
	}
	if v.k == valBool && v.b {
		return 1
	}
	return 0
}

func (p *parser) parseUnary(fields map[string]any) (val, error) {
	if p.peek().kind == tokOp && p.peek().text == "-" {
		p.next()
		v, err := p.parseUnary(fields)
		if err != nil {
			return val{}, err
		}
		if v.k == valStr {
			return val{}, fmt.Errorf("compute: unary minus needs a number")
		}
		return val{k: valNum, n: -numOrZero(v)}, nil
	}
	return p.parsePrimary(fields)
}

func (p *parser) parsePrimary(fields map[string]any) (val, error) {
	t := p.next()
	switch t.kind {
	case tokNumber:
		return val{k: valNum, n: t.num}, nil
	case tokString:
		return val{k: valStr, s: t.text}, nil
	case tokLParen:
		v, err := p.parseOr(fields)
		if err != nil {
			return val{}, err
		}
		if p.peek().kind != tokRParen {
			return val{}, fmt.Errorf("compute: missing )")
		}
		p.next()
		return v, nil
	case tokIdent:
		if p.peek().kind == tokLParen {
			return p.parseCall(t.text, fields)
		}
		p.noteIdent(t.text)
		if fields == nil {
			return val{}, nil
		}
		v, ok := fields[t.text]
		if !ok || v == nil {
			return val{}, nil
		}
		return valFrom(v), nil
	default:
		if t.text == "" {
			return val{}, fmt.Errorf("compute: unexpected end")
		}
		return val{}, fmt.Errorf("compute: unexpected %q", t.text)
	}
}

func (p *parser) parseCall(name string, fields map[string]any) (val, error) {
	p.next() // (
	var args []val
	if p.peek().kind != tokRParen {
		for {
			arg, err := p.parseOr(fields)
			if err != nil {
				return val{}, err
			}
			args = append(args, arg)
			if p.peek().kind != tokComma {
				break
			}
			p.next()
		}
	}
	if p.peek().kind != tokRParen {
		return val{}, fmt.Errorf("compute: missing ) after %s", name)
	}
	p.next()
	switch name {
	case "len":
		if len(args) != 1 {
			return val{}, fmt.Errorf("compute: len takes one argument")
		}
		return val{k: valNum, n: float64(utf8.RuneCountInString(args[0].display()))}, nil
	case "lower":
		if len(args) != 1 {
			return val{}, fmt.Errorf("compute: lower takes one argument")
		}
		return val{k: valStr, s: strings.ToLower(args[0].display())}, nil
	case "upper":
		if len(args) != 1 {
			return val{}, fmt.Errorf("compute: upper takes one argument")
		}
		return val{k: valStr, s: strings.ToUpper(args[0].display())}, nil
	case "trim":
		if len(args) != 1 {
			return val{}, fmt.Errorf("compute: trim takes one argument")
		}
		return val{k: valStr, s: strings.TrimSpace(args[0].display())}, nil
	case "coalesce":
		if len(args) == 0 {
			return val{}, fmt.Errorf("compute: coalesce needs an argument")
		}
		for _, a := range args {
			if a.k != valNull {
				return a, nil
			}
		}
		return val{}, nil
	default:
		return val{}, fmt.Errorf("compute: unknown function %s", name)
	}
}
