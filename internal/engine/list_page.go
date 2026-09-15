package engine

import (
	"context"
	"fmt"
	"strings"

	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"

	"github.com/graphql-go/graphql"
)

// ListPageOpts controls server-side list pagination and search.
// PageSize <= 0 means return the full filtered list (no LIMIT).
type ListPageOpts struct {
	Page     int
	PageSize int
	// Domain is a JSON Odoo-style domain (supports & | ! and leaf triples).
	Domain string
	// Q is a quick-search string matched with ILIKE across SearchIn fields.
	Q string
	// SearchIn limits Q to these field names; empty → all string-ish model fields.
	SearchIn []string
	// GroupBy field names (server returns Groups when requested via Groups()).
	GroupBy []string
}

// ListPageResult is a single page of model records plus the unpaged total.
type ListPageResult struct {
	Items    []Record
	Total    int
	Page     int
	PageSize int
}

// GroupBucket is one groupBy facet value.
type GroupBucket struct {
	Values []string `json:"values"`
	Count  int      `json:"count"`
}

// Normalize clamps page/pageSize to sane defaults.
func (o ListPageOpts) Normalize() ListPageOpts {
	if o.Page < 1 {
		o.Page = 1
	}
	if o.PageSize < 0 {
		o.PageSize = 0
	}
	return o
}

func (o ListPageOpts) hasFilter() bool {
	return strings.TrimSpace(o.Domain) != "" || strings.TrimSpace(o.Q) != ""
}

func parseListPageOpts(args map[string]any) ListPageOpts {
	opts := ListPageOpts{Page: 1, PageSize: 0}
	if v, ok := intArg(args["page"]); ok && v > 0 {
		opts.Page = v
	}
	if v, ok := intArg(args["pageSize"]); ok && v > 0 {
		opts.PageSize = v
	}
	if s, ok := args["domain"].(string); ok {
		opts.Domain = strings.TrimSpace(s)
	}
	if s, ok := args["q"].(string); ok {
		opts.Q = strings.TrimSpace(s)
	}
	opts.SearchIn = stringSliceArg(args["searchIn"])
	opts.GroupBy = stringSliceArg(args["groupBy"])
	return opts.Normalize()
}

func stringSliceArg(v any) []string {
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s := strings.TrimSpace(fmt.Sprint(item))
			if s != "" && s != "<nil>" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func intArg(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func slicePage(items []Record, opts ListPageOpts) ListPageResult {
	opts = opts.Normalize()
	total := len(items)
	if opts.PageSize <= 0 {
		return ListPageResult{Items: items, Total: total, Page: 1, PageSize: total}
	}
	start := (opts.Page - 1) * opts.PageSize
	if start >= total {
		return ListPageResult{Items: []Record{}, Total: total, Page: opts.Page, PageSize: opts.PageSize}
	}
	end := start + opts.PageSize
	if end > total {
		end = total
	}
	return ListPageResult{
		Items:    items[start:end],
		Total:    total,
		Page:     opts.Page,
		PageSize: opts.PageSize,
	}
}

func sliceAnyPage(items []any, opts ListPageOpts) (out []any, total int) {
	opts = opts.Normalize()
	total = len(items)
	if opts.PageSize <= 0 {
		return items, total
	}
	start := (opts.Page - 1) * opts.PageSize
	if start >= total {
		return []any{}, total
	}
	end := start + opts.PageSize
	if end > total {
		end = total
	}
	return items[start:end], total
}

// listFilterArgs are domain/q args shared by Count (no pagination required).
func listFilterArgs() graphql.FieldConfigArgument {
	return graphql.FieldConfigArgument{
		"domain":   &graphql.ArgumentConfig{Type: graphql.String},
		"q":        &graphql.ArgumentConfig{Type: graphql.String},
		"searchIn": &graphql.ArgumentConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		"groupBy":  &graphql.ArgumentConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
	}
}

// listPageArgs are shared GraphQL arguments for list queries.
func listPageArgs() graphql.FieldConfigArgument {
	args := listFilterArgs()
	args["page"] = &graphql.ArgumentConfig{Type: graphql.Int, Description: "1-based page (with pageSize)"}
	args["pageSize"] = &graphql.ArgumentConfig{Type: graphql.Int, Description: "Rows per page; omit for full list"}
	return args
}

var groupBucketTypes = map[string]*graphql.Object{}

func groupBucketType(app string) *graphql.Object {
	if t, ok := groupBucketTypes[app]; ok {
		return t
	}
	t := graphql.NewObject(graphql.ObjectConfig{
		Name: pascal(app) + "GroupBucket",
		Fields: graphql.Fields{
			"values": &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
			"count":  &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
	})
	groupBucketTypes[app] = t
	return t
}

func (s *modelService) ListPage(ctx context.Context, orgID string, opts ListPageOpts) (ListPageResult, error) {
	opts = opts.Normalize()
	expr, err := acl.ParseDomainExpr(opts.Domain)
	if err != nil {
		return ListPageResult{}, err
	}

	// Prefer SQL when we can compile filters and ACL allows it.
	if canSQLPage(ctx, s, orgID) {
		return s.listFilteredPage(ctx, orgID, opts, expr)
	}

	list, err := s.List(ctx, orgID)
	if err != nil {
		return ListPageResult{}, err
	}
	if list == nil {
		list = []Record{}
	}
	list = filterRecords(list, expr, opts, s.model)
	return slicePage(list, opts), nil
}

func (s *modelService) Count(ctx context.Context, orgID string) (int, error) {
	return s.CountOpts(ctx, orgID, ListPageOpts{})
}

func (s *modelService) CountOpts(ctx context.Context, orgID string, opts ListPageOpts) (int, error) {
	opts = opts.Normalize()
	page, err := s.ListPage(ctx, orgID, ListPageOpts{
		Page: 1, PageSize: 1,
		Domain: opts.Domain, Q: opts.Q, SearchIn: opts.SearchIn,
	})
	if err != nil {
		return 0, err
	}
	return page.Total, nil
}

func (s *modelService) Groups(ctx context.Context, orgID string, opts ListPageOpts) ([]GroupBucket, error) {
	opts = opts.Normalize()
	if len(opts.GroupBy) == 0 {
		return nil, nil
	}
	for _, f := range opts.GroupBy {
		if _, err := s.columnFor(f); err != nil {
			return nil, fmt.Errorf("groupBy: %w", err)
		}
	}
	// Load filtered set (no page slice) then aggregate in memory — correct with ACL masking.
	page, err := s.ListPage(ctx, orgID, ListPageOpts{
		Domain: opts.Domain, Q: opts.Q, SearchIn: opts.SearchIn, PageSize: 0,
	})
	if err != nil {
		return nil, err
	}
	counts := map[string]*GroupBucket{}
	order := []string{}
	for _, rec := range page.Items {
		vals := make([]string, len(opts.GroupBy))
		for i, f := range opts.GroupBy {
			vals[i] = fmt.Sprint(rec[f])
			if vals[i] == "<nil>" {
				vals[i] = ""
			}
		}
		key := strings.Join(vals, "\x1f")
		if b, ok := counts[key]; ok {
			b.Count++
		} else {
			b := &GroupBucket{Values: vals, Count: 1}
			counts[key] = b
			order = append(order, key)
		}
	}
	out := make([]GroupBucket, 0, len(order))
	for _, k := range order {
		out = append(out, *counts[k])
	}
	return out, nil
}

func canSQLPage(ctx context.Context, s *modelService, orgID string) bool {
	if s.skipACL(ctx) {
		return true
	}
	authz := s.authorizer()
	if authz == nil {
		return true
	}
	filter, err := authz.ListDomain(ctx, orgID, s.resourceName(), acl.ActRead)
	if err != nil {
		return false
	}
	return filter.Unrestricted || filter.DenyAll
}

func (s *modelService) listFilteredPage(ctx context.Context, orgID string, opts ListPageOpts, expr *acl.DomainExpr) (ListPageResult, error) {
	opts = opts.Normalize()
	if authz := s.authorizer(); authz != nil && !s.skipACL(ctx) {
		filter, err := authz.ListDomain(ctx, orgID, s.resourceName(), acl.ActRead)
		if err == nil && filter.DenyAll {
			return ListPageResult{Items: []Record{}, Total: 0, Page: opts.Page, PageSize: opts.PageSize}, nil
		}
	}

	where, args, err := s.buildFilterSQL(orgID, opts, expr, 2)
	if err != nil {
		// Fall back to list+filter if domain fields cannot be compiled.
		list, lerr := s.List(ctx, orgID)
		if lerr != nil {
			return ListPageResult{}, lerr
		}
		list = filterRecords(list, expr, opts, s.model)
		return slicePage(list, opts), nil
	}

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, s.qtable(), where)
	var total int
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListPageResult{}, err
	}

	order := "ORDER BY updated_at DESC"
	if len(opts.GroupBy) > 0 {
		cols := make([]string, 0, len(opts.GroupBy))
		for _, f := range opts.GroupBy {
			col, err := s.columnFor(f)
			if err != nil {
				return ListPageResult{}, err
			}
			cols = append(cols, quoteIdent(col))
		}
		order = "ORDER BY " + strings.Join(cols, ", ") + ", updated_at DESC"
	}

	sql := fmt.Sprintf(`SELECT %s FROM %s WHERE %s %s`, s.selectList(), s.qtable(), where, order)
	queryArgs := append([]any{}, args...)
	if opts.PageSize > 0 {
		offset := (opts.Page - 1) * opts.PageSize
		sql += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(queryArgs)+1, len(queryArgs)+2)
		queryArgs = append(queryArgs, opts.PageSize, offset)
	}

	rows, err := s.pool.Query(ctx, sql, queryArgs...)
	if err != nil {
		return ListPageResult{}, err
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		rec, err := s.scan(rows)
		if err != nil {
			return ListPageResult{}, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return ListPageResult{}, err
	}
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = total
	}
	return ListPageResult{Items: out, Total: total, Page: opts.Page, PageSize: pageSize}, nil
}

func (s *modelService) buildFilterSQL(orgID string, opts ListPageOpts, expr *acl.DomainExpr, startArg int) (string, []any, error) {
	parts := []string{fmt.Sprintf("org_id = $1 AND deleted = false")}
	args := []any{orgID}
	arg := startArg

	if expr != nil {
		frag, next, err := acl.CompileDomainExpr(expr, s.colQuote, arg, acl.PrincipalContext{})
		if err != nil {
			return "", nil, err
		}
		arg = next
		if frag.Clause != "" {
			parts = append(parts, "("+frag.Clause+")")
			args = append(args, frag.Args...)
		}
	}

	q := strings.TrimSpace(opts.Q)
	if q != "" {
		fields := opts.SearchIn
		if len(fields) == 0 {
			fields = stringishFields(s.model)
		}
		var orParts []string
		pattern := q
		if !strings.Contains(pattern, "%") {
			pattern = "%" + pattern + "%"
		}
		for _, f := range fields {
			col, err := s.columnFor(f)
			if err != nil {
				continue
			}
			orParts = append(orParts, fmt.Sprintf("%s::text ILIKE $%d", quoteIdent(col), arg))
		}
		if len(orParts) == 0 {
			return "", nil, fmt.Errorf("q: no searchable fields")
		}
		parts = append(parts, "("+strings.Join(orParts, " OR ")+")")
		args = append(args, pattern)
		arg++
	}

	_ = arg
	return strings.Join(parts, " AND "), args, nil
}

func stringishFields(model appspec.ModelSpec) []string {
	out := []string{}
	for _, f := range model.Fields {
		switch f.CanonicalType() {
		case appspec.TypeString, appspec.TypeText, appspec.TypeEnum:
			out = append(out, f.Name)
		}
	}
	return out
}

func filterRecords(list []Record, expr *acl.DomainExpr, opts ListPageOpts, model appspec.ModelSpec) []Record {
	q := strings.TrimSpace(opts.Q)
	fields := opts.SearchIn
	if len(fields) == 0 {
		fields = stringishFields(model)
	}
	out := make([]Record, 0, len(list))
	for _, rec := range list {
		if expr != nil && !expr.Match(rec, acl.PrincipalContext{}) {
			continue
		}
		if q != "" {
			ql := strings.ToLower(q)
			ok := false
			for _, f := range fields {
				if strings.Contains(strings.ToLower(fmt.Sprint(rec[f])), ql) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, rec)
	}
	return out
}
