package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"kaizengo/packages/sdk-go/appspec"
)

// computeSkip stops a related lookup from recomputing the row it just loaded.
type computeSkip struct{}

// relatedHop loads the target of one many2one hop.
type relatedHop func(ctx context.Context, ref appspec.RelationRef, id string) (Record, appspec.ModelSpec, error)

func (s *modelService) stampComputes(ctx context.Context, orgID string, payload map[string]any, prev Record) error {
	merged := map[string]any{}
	for k, v := range prev {
		merged[k] = v
	}
	for k, v := range payload {
		merged[k] = v
	}
	if err := applyFieldComputes(ctx, s.spec, s.model, merged, prev, orgID, s.hooks, true, s.hop(orgID)); err != nil {
		return err
	}
	for _, f := range s.model.Fields {
		if f.Computed() && f.StoreValue() {
			payload[f.Name] = merged[f.Name]
		}
	}
	return nil
}

func (s *modelService) fillComputed(ctx context.Context, recs []Record) error {
	if ctx.Value(computeSkip{}) != nil {
		return nil
	}
	for _, rec := range recs {
		orgID := str(rec["orgId"])
		if err := applyFieldComputes(ctx, s.spec, s.model, rec, rec, orgID, s.hooks, false, s.hop(orgID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *modelService) hop(orgID string) relatedHop {
	return func(ctx context.Context, ref appspec.RelationRef, id string) (Record, appspec.ModelSpec, error) {
		ctx = context.WithValue(ctx, computeSkip{}, true)
		var reg *ModelRegistry
		if ref.App == "" || ref.App == s.spec.Name {
			reg = s.registry
		} else if s.host != nil {
			var err error
			reg, err = ModelsFromHost(s.host, ref.App)
			if err != nil {
				return nil, appspec.ModelSpec{}, err
			}
		} else {
			return nil, appspec.ModelSpec{}, fmt.Errorf("related app %s is not loaded", ref.App)
		}
		if reg == nil {
			return nil, appspec.ModelSpec{}, fmt.Errorf("model registry unavailable")
		}
		svc, err := reg.require(ref.Model)
		if err != nil {
			return nil, appspec.ModelSpec{}, err
		}
		rec, err := svc.getRaw(ctx, id)
		if err != nil {
			if errors.Is(err, errNotFound) {
				return nil, appspec.ModelSpec{}, nil
			}
			return nil, appspec.ModelSpec{}, err
		}
		if boolVal(rec["deleted"]) || (orgID != "" && str(rec["orgId"]) != orgID) {
			return nil, svc.model, nil
		}
		return rec, svc.model, nil
	}
}

func applyFieldComputes(ctx context.Context, app appspec.AppSpec, model appspec.ModelSpec, fields map[string]any, prev Record, orgID string, hooks Hooks, storedOnly bool, hop relatedHop) error {
	order, err := appspec.OrderedComputes(model.Name, model.Fields)
	if err != nil {
		return err
	}
	for _, f := range order {
		if storedOnly && !f.StoreValue() {
			continue
		}
		if !storedOnly && f.StoreValue() {
			continue
		}
		var raw any
		switch {
		case f.HookCompute():
			fn := hooks.Compute[f.Name]
			if fn == nil {
				return fmt.Errorf("field %s: no compute hook registered", f.Name)
			}
			raw, err = fn(HookContext{
				Context: ctx, App: app, Model: model,
				OrgID: orgID, RecordID: str(fields["id"]), Fields: fields, Record: prev,
			})
			if err != nil {
				return err
			}
		case strings.TrimSpace(f.Related) != "":
			raw, err = walkRelated(ctx, app.Name, model, fields, f.Related, hop)
			if err != nil {
				return fmt.Errorf("field %s: %w", f.Name, err)
			}
		default:
			raw, err = appspec.EvalCompute(f.Compute, fields)
			if err != nil {
				return fmt.Errorf("field %s: %w", f.Name, err)
			}
		}
		if raw == nil {
			raw = fieldZero(f)
		}
		coerced, err := coerceStoredValue(f, raw)
		if err != nil {
			return fmt.Errorf("field %s: %w", f.Name, err)
		}
		fields[f.Name] = coerced
	}
	return nil
}

func walkRelated(ctx context.Context, app string, model appspec.ModelSpec, fields map[string]any, path string, hop relatedHop) (any, error) {
	parts := strings.Split(path, ".")
	curModel := model
	cur := fields
	curApp := app
	for i, part := range parts {
		if i == len(parts)-1 {
			if cur == nil {
				return nil, nil
			}
			return cur[part], nil
		}
		f, ok := fieldByName(curModel, part)
		if !ok || f.CanonicalType() != appspec.TypeMany2One {
			return nil, fmt.Errorf("related %s: %s is not many2one", path, part)
		}
		id := strings.TrimSpace(str(cur[part]))
		if id == "" || id == "<nil>" {
			return nil, nil
		}
		ref, ok := f.ResolveRelation(curApp)
		if !ok {
			return nil, fmt.Errorf("related %s: %s has no target", path, part)
		}
		if hop == nil {
			return nil, fmt.Errorf("related %s: lookup is unavailable", path)
		}
		rec, next, err := hop(ctx, ref, id)
		if err != nil || rec == nil {
			return nil, err
		}
		cur = rec
		curModel = next
		curApp = ref.App
	}
	return nil, nil
}
