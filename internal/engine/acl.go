package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"kaizengo/internal/auth"
	"kaizengo/packages/sdk-go/acl"
)

// modelResource is the default ACL resource id for a spec model: "{app}.{model}".
func modelResource(app, model string) string {
	return app + "." + model
}

func (s *modelService) resourceName() string {
	if s.spec.Resource != "" && s.spec.Resource != s.spec.Name {
		// Explicit app-level resource still scopes models as app.model unless
		// we only have app-level; prefer per-model.
	}
	return modelResource(s.spec.Name, s.model.Name)
}

func (s *modelService) authorizer() acl.Authorizer {
	if s.host == nil {
		return nil
	}
	raw, ok := s.host.Lookup(acl.ServiceName)
	if !ok {
		return nil
	}
	authz, ok := raw.(acl.Authorizer)
	if !ok {
		return nil
	}
	return authz
}

func (s *modelService) skipACL(ctx context.Context) bool {
	return IsInternal(ctx) || s.authorizer() == nil
}

func (s *modelService) maskRead(ctx context.Context, orgID string, rec Record) (Record, error) {
	authz := s.authorizer()
	if authz == nil || rec == nil {
		return rec, nil
	}
	res := s.resourceName()
	d, err := authz.Can(ctx, acl.Check{OrgID: orgID, Resource: res, Action: acl.ActRead, Record: rec})
	if err != nil {
		return nil, err
	}
	if !d.Allowed {
		return nil, auth.ErrForbidden
	}
	denied, err := authz.DeniedFields(ctx, orgID, res, acl.ActRead, rec)
	if err != nil {
		return nil, err
	}
	return Record(acl.MaskRecord(rec, d.FieldsRead, denied)), nil
}

func (s *modelService) enforceWrite(ctx context.Context, orgID string, action string, record Record, fields map[string]any) (map[string]any, error) {
	authz := s.authorizer()
	if authz == nil {
		return fields, nil
	}
	res := s.resourceName()
	touch := make([]string, 0, len(fields))
	for k := range fields {
		touch = append(touch, k)
	}
	check := acl.Check{OrgID: orgID, Resource: res, Action: action, Record: record, Fields: touch}
	d, err := authz.Can(ctx, check)
	if err != nil {
		return nil, err
	}
	if !d.Allowed {
		return nil, auth.ErrForbidden
	}
	denied, err := authz.DeniedFields(ctx, orgID, res, action, record)
	if err != nil {
		return nil, err
	}
	clean, forbidden := acl.FilterWriteFields(fields, d.FieldsWrite, denied)
	if len(forbidden) > 0 {
		return nil, fmt.Errorf("%w: fields %s", auth.ErrForbidden, strings.Join(forbidden, ", "))
	}
	return clean, nil
}

func (s *modelService) colQuote(field string) (string, bool) {
	col, err := s.columnFor(field)
	if err != nil {
		return "", false
	}
	return quoteIdent(col), true
}

func (s *modelService) listWithACL(ctx context.Context, orgID string) ([]Record, error) {
	authz := s.authorizer()
	if authz == nil {
		return s.listRaw(ctx, orgID)
	}
	res := s.resourceName()
	// Model-level read grant
	d, err := authz.Can(ctx, acl.Check{OrgID: orgID, Resource: res, Action: acl.ActRead})
	if err != nil {
		return nil, err
	}
	if !d.Allowed {
		return nil, auth.ErrForbidden
	}
	filter, err := authz.ListDomain(ctx, orgID, res, acl.ActRead)
	if err != nil {
		return nil, err
	}
	if filter.DenyAll {
		return []Record{}, nil
	}

	p, _ := auth.PrincipalFrom(ctx)
	pc := acl.PrincipalContext{}
	if p != nil {
		pc.UserID, pc.OrgID = p.UserID, p.OrgID
	}

	frag, err := acl.CompileListFilter(filter, s.colQuote, 2, pc)
	if err != nil {
		// fall back to post-filter
		return s.listPostFilter(ctx, orgID, authz, res, filter, pc)
	}

	sql := fmt.Sprintf(`SELECT %s FROM %s WHERE org_id = $1 AND deleted = false`, s.selectList(), s.qtable())
	args := []any{orgID}
	if frag.Clause != "" {
		sql += " AND (" + frag.Clause + ")"
		args = append(args, frag.Args...)
	}
	sql += " ORDER BY updated_at DESC"

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return s.listPostFilter(ctx, orgID, authz, res, filter, pc)
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		rec, err := s.scan(rows)
		if err != nil {
			return nil, err
		}
		masked, err := s.maskRead(ctx, orgID, rec)
		if err != nil {
			if errors.Is(err, auth.ErrForbidden) {
				continue
			}
			return nil, err
		}
		out = append(out, masked)
	}
	return out, rows.Err()
}

func (s *modelService) listPostFilter(ctx context.Context, orgID string, authz acl.Authorizer, res string, filter acl.ListFilter, pc acl.PrincipalContext) ([]Record, error) {
	raw, err := s.listRaw(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]Record, 0, len(raw))
	for _, rec := range raw {
		if !recordPassesFilter(rec, filter, pc) {
			continue
		}
		masked, err := s.maskRead(ctx, orgID, rec)
		if err != nil {
			if errors.Is(err, auth.ErrForbidden) {
				continue
			}
			return nil, err
		}
		out = append(out, masked)
	}
	return out, nil
}

func recordPassesFilter(rec Record, filter acl.ListFilter, pc acl.PrincipalContext) bool {
	if filter.DenyAll {
		return false
	}
	if filter.Unrestricted {
		return true
	}
	for _, d := range filter.Deny {
		if d.Match(rec, pc) {
			return false
		}
	}
	if len(filter.Allow) == 0 {
		return true
	}
	for _, a := range filter.Allow {
		if a.Match(rec, pc) {
			return true
		}
	}
	return false
}

func (s *modelService) listRaw(ctx context.Context, orgID string) ([]Record, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM %s WHERE org_id = $1 AND deleted = false ORDER BY updated_at DESC
	`, s.selectList(), s.qtable()), orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		rec, err := s.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
