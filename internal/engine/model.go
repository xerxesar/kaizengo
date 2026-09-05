package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	"kaizengo/packages/sdk-go/appspec"
	"kaizengo/internal/events"
	"kaizengo/internal/events/pgstore"
	"kaizengo/packages/sdk-go/i18n"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errNotFound = errors.New("record not found")
	errRequired = errors.New("required field missing")
)

// Record is a projected document for a declarative model.
type Record map[string]any

type modelService struct {
	store    *pgstore.Store
	pool     *pgxpool.Pool
	schema   string
	spec     appspec.AppSpec
	model    appspec.ModelSpec
	hooks    Hooks
	registry *ModelRegistry
	host     *module.Host
}

func newModelService(store *pgstore.Store, spec appspec.AppSpec, model appspec.ModelSpec, registry *HookRegistry) *modelService {
	return &modelService{
		store:  store,
		pool:   store.Pool(),
		schema: spec.Schema,
		spec:   spec,
		model:  model,
		hooks:  registry.forModel(model.Name),
	}
}

func (s *modelService) List(ctx context.Context, orgID string) ([]Record, error) {
	if s.skipACL(ctx) {
		return s.listRaw(ctx, orgID)
	}
	return s.listWithACL(ctx, orgID)
}

func (s *modelService) listAll(ctx context.Context) ([]Record, error) {
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM %s WHERE deleted = false ORDER BY created_at
	`, s.selectList(), s.qtable()))
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

func (s *modelService) findBy(ctx context.Context, field, value string) (Record, error) {
	col, err := s.columnFor(field)
	if err != nil {
		return nil, err
	}
	row := s.pool.QueryRow(ctx, fmt.Sprintf(
		`SELECT %s FROM %s WHERE %s = $1 AND deleted = false LIMIT 1`,
		s.selectList(), s.qtable(), quoteIdent(col),
	), value)
	rec, err := s.scan(row)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, errNotFound
		}
		return nil, err
	}
	return rec, nil
}

func (s *modelService) listBy(ctx context.Context, orgID, field, value string) ([]Record, error) {
	col, err := s.columnFor(field)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT %s FROM %s WHERE org_id = $1 AND %s = $2 AND deleted = false ORDER BY updated_at DESC
	`, s.selectList(), s.qtable(), quoteIdent(col)), orgID, value)
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

func (s *modelService) columnFor(field string) (string, error) {
	switch field {
	case "id", "orgId", "authorId":
		return colName(field), nil
	}
	for _, f := range s.model.Fields {
		if f.Name == field {
			return colName(f.Name), nil
		}
	}
	return "", fmt.Errorf("unknown field %q on model %q", field, s.model.Name)
}

func (s *modelService) Get(ctx context.Context, orgID, id string) (Record, error) {
	rec, err := s.getRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if str(rec["orgId"]) != orgID || boolVal(rec["deleted"]) {
		return nil, errNotFound
	}
	if s.skipACL(ctx) {
		return rec, nil
	}
	return s.maskRead(ctx, orgID, rec)
}

func (s *modelService) rejectExternalWrite(ctx context.Context) error {
	if !s.model.Internal || IsInternal(ctx) {
		return nil
	}
	return i18n.Error(s.spec.Name + ".error." + s.model.Name + ".internal")
}

func (s *modelService) Create(ctx context.Context, orgID, authorID string, fields map[string]any) (Record, error) {
	if err := s.rejectExternalWrite(ctx); err != nil {
		return nil, err
	}
	if !s.skipACL(ctx) {
		cleaned, err := s.enforceWrite(ctx, orgID, acl.ActCreate, nil, fields)
		if err != nil {
			return nil, err
		}
		fields = cleaned
	}
	id := uuid.NewString()
	if v, ok := fields["id"].(string); ok && strings.TrimSpace(v) != "" {
		id = strings.TrimSpace(v)
	}
	payload, err := s.normalizeFields(fields, true)
	if err != nil {
		return nil, err
	}
	payload["id"] = id
	payload["orgId"] = orgID
	payload["authorId"] = authorID

	hc := HookContext{
		Context: ctx, App: s.spec, Model: s.model,
		OrgID: orgID, UserID: authorID, RecordID: id, Fields: payload,
	}
	if err := s.runHook(s.hooks.BeforeCreate, hc); err != nil {
		return nil, err
	}
	if err := runExtensions(modelPoint(s.spec.Name, s.model.Name, "beforeCreate"), hc, true); err != nil {
		return nil, err
	}
	if err := validateSpecFields(s.model, hc.Fields); err != nil {
		return nil, err
	}

	evs, err := s.store.Append(ctx, id, s.model.Stream, 0, events.NewEvent{
		Type:    eventCreated(s.spec, s.model),
		Payload: hc.Fields,
	})
	if err != nil {
		return nil, err
	}
	if err := s.project(ctx, evs...); err != nil {
		return nil, err
	}
	// Load without ACL mask for hooks; mask on return for external callers.
	rec, err := s.getRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if str(rec["orgId"]) != orgID || boolVal(rec["deleted"]) {
		return nil, errNotFound
	}
	hc.Record = rec
	_ = s.runHook(s.hooks.AfterCreate, hc)
	_ = runExtensions(modelPoint(s.spec.Name, s.model.Name, "afterCreate"), hc, false)
	syncSearchIndex(ctx, s.spec, s.model, orgID, rec, false)
	if s.skipACL(ctx) {
		return rec, nil
	}
	return s.maskRead(ctx, orgID, rec)
}

func (s *modelService) Update(ctx context.Context, orgID, id string, fields map[string]any) (Record, error) {
	if err := s.rejectExternalWrite(ctx); err != nil {
		return nil, err
	}
	rec, err := s.getRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if str(rec["orgId"]) != orgID || boolVal(rec["deleted"]) {
		return nil, errNotFound
	}
	if !s.skipACL(ctx) {
		// record-level + field write check
		authz := s.authorizer()
		res := s.resourceName()
		if err := authz.MustAllow(ctx, acl.Check{
			OrgID: orgID, Resource: res, Action: acl.ActUpdate, Record: rec,
		}); err != nil {
			return nil, err
		}
		cleaned, err := s.enforceWrite(ctx, orgID, acl.ActUpdate, rec, fields)
		if err != nil {
			return nil, err
		}
		fields = cleaned
	}
	payload, err := s.normalizeFields(fields, false)
	if err != nil {
		return nil, err
	}
	hc := HookContext{
		Context: ctx, App: s.spec, Model: s.model,
		OrgID: orgID, RecordID: id, Fields: payload, Record: rec,
	}
	if err := s.runHook(s.hooks.BeforeUpdate, hc); err != nil {
		return nil, err
	}
	if err := runExtensions(modelPoint(s.spec.Name, s.model.Name, "beforeUpdate"), hc, true); err != nil {
		return nil, err
	}
	if err := validateSpecFields(s.model, hc.Fields); err != nil {
		return nil, err
	}

	agg, err := s.loadVersion(ctx, id)
	if err != nil {
		return nil, err
	}
	evs, err := s.store.Append(ctx, id, s.model.Stream, agg, events.NewEvent{
		Type:    eventUpdated(s.spec, s.model),
		Payload: hc.Fields,
	})
	if err != nil {
		return nil, err
	}
	if err := s.project(ctx, evs...); err != nil {
		return nil, err
	}
	updated, err := s.getRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	hc.Record = updated
	_ = s.runHook(s.hooks.AfterUpdate, hc)
	_ = runExtensions(modelPoint(s.spec.Name, s.model.Name, "afterUpdate"), hc, false)
	syncSearchIndex(ctx, s.spec, s.model, orgID, updated, false)
	if s.skipACL(ctx) {
		return updated, nil
	}
	return s.maskRead(ctx, orgID, updated)
}

func (s *modelService) Delete(ctx context.Context, orgID, id string) error {
	if err := s.rejectExternalWrite(ctx); err != nil {
		return err
	}
	rec, err := s.getRaw(ctx, id)
	if err != nil {
		return err
	}
	if str(rec["orgId"]) != orgID || boolVal(rec["deleted"]) {
		return errNotFound
	}
	if !s.skipACL(ctx) {
		authz := s.authorizer()
		if err := authz.MustAllow(ctx, acl.Check{
			OrgID: orgID, Resource: s.resourceName(), Action: acl.ActDelete, Record: rec,
		}); err != nil {
			return err
		}
	}
	hc := HookContext{
		Context: ctx, App: s.spec, Model: s.model,
		OrgID: orgID, RecordID: id, Record: rec,
	}
	if err := s.runHook(s.hooks.BeforeDelete, hc); err != nil {
		return err
	}
	if err := runExtensions(modelPoint(s.spec.Name, s.model.Name, "beforeDelete"), hc, true); err != nil {
		return err
	}

	agg, err := s.loadVersion(ctx, id)
	if err != nil {
		return err
	}
	evs, err := s.store.Append(ctx, id, s.model.Stream, agg, events.NewEvent{
		Type:    eventDeleted(s.spec, s.model),
		Payload: map[string]any{},
	})
	if err != nil {
		return err
	}
	if err := s.project(ctx, evs...); err != nil {
		return err
	}
	_ = s.runHook(s.hooks.AfterDelete, hc)
	_ = runExtensions(modelPoint(s.spec.Name, s.model.Name, "afterDelete"), hc, false)
	syncSearchIndex(ctx, s.spec, s.model, orgID, rec, true)
	return nil
}

func (s *modelService) runHook(fn func(HookContext) error, hc HookContext) error {
	if fn == nil {
		return nil
	}
	return fn(hc)
}

func (s *modelService) normalizeFields(in map[string]any, requireAll bool) (map[string]any, error) {
	out := map[string]any{}
	for _, f := range s.model.Fields {
		if f.Readonly && !requireAll {
			// Updates may not change readonly fields.
			continue
		}
		v, ok := in[f.Name]
		missing := !ok || v == nil
		if missing || (fmt.Sprint(v) == "" && f.Default != nil) {
			if requireAll && f.Default != nil {
				v = f.Default
			} else if requireAll && f.Required && !f.Readonly {
				return nil, fmt.Errorf("%w: %s", errRequired, f.Name)
			} else if requireAll && missing {
				v = fieldZero(f)
			} else if !requireAll {
				continue
			}
		}
		if f.Readonly && requireAll && f.Default != nil && missing {
			v = f.Default
		}
		switch {
		case f.CanonicalType() == appspec.TypeOne2Many:
			continue
		case f.IsEnum():
			sval := strings.TrimSpace(fmt.Sprint(v))
			if sval == "" && f.Required {
				return nil, fmt.Errorf("%w: %s", errRequired, f.Name)
			}
			if sval != "" && !enumContains(f.Values, sval) {
				return nil, fmt.Errorf("field %s must be one of %v", f.Name, f.Values)
			}
			out[f.Name] = sval
		default:
			nv, err := coerceStoredValue(f, v)
			if err != nil {
				return nil, err
			}
			if f.Required && isEmptyValue(f, nv) {
				return nil, fmt.Errorf("%w: %s", errRequired, f.Name)
			}
			out[f.Name] = nv
		}
	}
	return out, nil
}

func fieldZero(f appspec.FieldSpec) any {
	switch f.CanonicalType() {
	case appspec.TypeInt:
		return 0
	case appspec.TypeNumber:
		return 0.0
	case appspec.TypeBool:
		return false
	case appspec.TypeMany2Many:
		return []string{}
	default:
		return ""
	}
}

func enumContains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func (s *modelService) loadVersion(ctx context.Context, id string) (int64, error) {
	evs, err := s.store.LoadStream(ctx, id)
	if err != nil {
		if errors.Is(err, events.ErrNotFound) {
			return 0, errNotFound
		}
		return 0, err
	}
	if len(evs) == 0 {
		return 0, errNotFound
	}
	return evs[len(evs)-1].Version, nil
}

func (s *modelService) project(ctx context.Context, evs ...events.Event) error {
	for _, ev := range evs {
		switch ev.Type {
		case eventCreated(s.spec, s.model):
			var p map[string]any
			if err := json.Unmarshal(ev.Payload, &p); err != nil {
				return err
			}
			cols := []string{"id", "org_id", "author_id", "deleted", "created_at", "updated_at"}
			args := []any{str(p["id"]), str(p["orgId"]), str(p["authorId"]), false, ev.OccurredAt, ev.OccurredAt}
			for _, f := range s.model.Fields {
				if !f.Stored() {
					continue
				}
				cols = append(cols, colName(f.Name))
				args = append(args, projectValue(f, p[f.Name]))
			}
			placeholders := make([]string, len(args))
			for i := range args {
				placeholders[i] = fmt.Sprintf("$%d", i+1)
			}
			setters := []string{"updated_at = excluded.updated_at"}
			for _, f := range s.model.Fields {
				if !f.Stored() {
					continue
				}
				setters = append(setters, fmt.Sprintf("%s = excluded.%s", quoteIdent(colName(f.Name)), quoteIdent(colName(f.Name))))
			}
			_, err := s.pool.Exec(ctx, fmt.Sprintf(`
				INSERT INTO %s (%s) VALUES (%s)
				ON CONFLICT(id) DO UPDATE SET %s
			`, s.qtable(), joinIdents(cols), strings.Join(placeholders, ","), strings.Join(setters, ", ")), args...)
			if err != nil {
				return err
			}
		case eventUpdated(s.spec, s.model):
			var p map[string]any
			if err := json.Unmarshal(ev.Payload, &p); err != nil {
				return err
			}
			sets := []string{"updated_at = $2"}
			args := []any{ev.StreamID, ev.OccurredAt}
			i := 3
			for _, f := range s.model.Fields {
				if _, ok := p[f.Name]; !ok || !f.Stored() {
					continue
				}
				sets = append(sets, fmt.Sprintf("%s = $%d", quoteIdent(colName(f.Name)), i))
				args = append(args, projectValue(f, p[f.Name]))
				i++
			}
			_, err := s.pool.Exec(ctx, fmt.Sprintf(`UPDATE %s SET %s WHERE id = $1`, s.qtable(), strings.Join(sets, ", ")), args...)
			if err != nil {
				return err
			}
		case eventDeleted(s.spec, s.model):
			_, err := s.pool.Exec(ctx, fmt.Sprintf(`
				UPDATE %s SET deleted = true, updated_at = $2 WHERE id = $1
			`, s.qtable()), ev.StreamID, ev.OccurredAt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *modelService) getRaw(ctx context.Context, id string) (Record, error) {
	row := s.pool.QueryRow(ctx, fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1`, s.selectList(), s.qtable()), id)
	rec, err := s.scan(row)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return nil, errNotFound
		}
		return nil, err
	}
	return rec, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func (s *modelService) scan(row scanner) (Record, error) {
	var (
		id, orgID, authorID  string
		deleted              bool
		createdAt, updatedAt time.Time
	)
	dest := []any{&id, &orgID, &authorID, &deleted, &createdAt, &updatedAt}
	stored := storedFields(s.model)
	fieldPtrs := make([]any, len(stored))
	for i, f := range stored {
		fieldPtrs[i] = scanPtr(f)
		dest = append(dest, fieldPtrs[i])
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	rec := Record{
		"id":        id,
		"orgId":     orgID,
		"authorId":  authorID,
		"deleted":   deleted,
		"createdAt": createdAt.UTC().Format(time.RFC3339),
		"updatedAt": updatedAt.UTC().Format(time.RFC3339),
	}
	for i, f := range stored {
		rec[f.Name] = derefScan(f, fieldPtrs[i])
	}
	return rec, nil
}

func (s *modelService) selectList() string {
	cols := []string{"id", "org_id", "author_id", "deleted", "created_at", "updated_at"}
	for _, f := range storedFields(s.model) {
		cols = append(cols, colName(f.Name))
	}
	return joinIdents(cols)
}

func (s *modelService) qtable() string {
	return quoteIdent(s.schema) + "." + quoteIdent(readTable(s.model))
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func joinIdents(cols []string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = quoteIdent(c)
	}
	return strings.Join(out, ", ")
}

func str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func boolVal(v any) bool {
	b, _ := v.(bool)
	return b
}
