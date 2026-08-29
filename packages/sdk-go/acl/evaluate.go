package acl

import (
	"sort"
	"strings"
)

// Evaluate computes a Decision from policy entries for a Check.
// Entries should already be filtered to the principal's roles; Active is still checked.
func Evaluate(entries []Entry, check Check, p PrincipalContext) Decision {
	matched := filterMatch(entries, check.Resource, check.Action)
	if len(matched) == 0 {
		return Decision{Allowed: false, FieldsRead: []string{}, FieldsWrite: []string{}}
	}
	return evaluateMatched(matched, check, p)
}

// EvaluateCatalog is like Evaluate but defaults to allowed when no policy matches.
// Used for menus/views where apps opt in with deny (or explicit allow) rules only.
func EvaluateCatalog(entries []Entry, check Check, p PrincipalContext) Decision {
	matched := filterMatch(entries, check.Resource, check.Action)
	if len(matched) == 0 {
		return Decision{Allowed: true}
	}
	return evaluateMatched(matched, check, p)
}

func evaluateMatched(matched []Entry, check Check, p PrincipalContext) Decision {
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority > matched[j].Priority
		}
		// deny before allow at same priority
		if matched[i].Effect != matched[j].Effect {
			return matched[i].Effect == EffectDeny
		}
		return matched[i].ID < matched[j].ID
	})

	// field -> effect; first covering applicable entry wins
	fieldEffect := map[string]string{}
	starEffect := ""
	hasAllow := false

	for _, e := range matched {
		if check.Record != nil && len(e.Domain) > 0 && !e.Domain.Match(check.Record, p) {
			continue
		}
		// Without a record, domain-restricted entries still count for create/model grant
		// only when domain is empty; otherwise skip for field-mask grants on create.
		if check.Record == nil && len(e.Domain) > 0 && check.Action != ActCreate {
			// List/get without record uses ListDomain separately; here skip domain rules.
			continue
		}
		if check.Record == nil && len(e.Domain) > 0 && check.Action == ActCreate {
			continue
		}

		fields := e.Fields
		if len(fields) == 0 {
			fields = []string{FieldsAll}
		}
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if f == FieldsAll {
				if starEffect == "" {
					starEffect = e.Effect
				}
				if e.Effect == EffectAllow {
					hasAllow = true
				}
				continue
			}
			if _, ok := fieldEffect[f]; ok {
				continue
			}
			fieldEffect[f] = e.Effect
			if e.Effect == EffectAllow {
				hasAllow = true
			}
		}
	}

	if starEffect == EffectAllow {
		hasAllow = true
	}

	allowedField := func(name string) bool {
		if eff, ok := fieldEffect[name]; ok {
			return eff == EffectAllow
		}
		if starEffect != "" {
			return starEffect == EffectAllow
		}
		return false
	}

	if len(check.Fields) > 0 {
		for _, f := range check.Fields {
			if !allowedField(f) {
				return Decision{Allowed: false, FieldsRead: []string{}, FieldsWrite: []string{}}
			}
		}
		return Decision{
			Allowed:     true,
			FieldsRead:  maskFrom(fieldEffect, starEffect, true),
			FieldsWrite: maskFrom(fieldEffect, starEffect, true),
		}
	}

	if !hasAllow && starEffect != EffectAllow {
		// no allow covering anything
		if starEffect == EffectDeny {
			return Decision{Allowed: false, FieldsRead: []string{}, FieldsWrite: []string{}}
		}
		if len(fieldEffect) == 0 {
			return Decision{Allowed: false, FieldsRead: []string{}, FieldsWrite: []string{}}
		}
		// only field-level denies/allows — allowed if any allow field exists
		any := false
		for _, eff := range fieldEffect {
			if eff == EffectAllow {
				any = true
				break
			}
		}
		if !any {
			return Decision{Allowed: false, FieldsRead: []string{}, FieldsWrite: []string{}}
		}
	}

	readMask := maskFrom(fieldEffect, starEffect, false)
	writeMask := readMask
	return Decision{
		Allowed:     true,
		FieldsRead:  readMask,
		FieldsWrite: writeMask,
	}
}

// maskFrom returns nil for all fields, or the explicit allow list.
func maskFrom(fieldEffect map[string]string, starEffect string, requireAllTouch bool) []string {
	_ = requireAllTouch
	if starEffect == EffectAllow {
		// subtract explicit denies
		var denies []string
		for f, eff := range fieldEffect {
			if eff == EffectDeny {
				denies = append(denies, f)
			}
		}
		if len(denies) == 0 {
			return nil // all
		}
		// cannot express "all except" as nil; return special: nil means all,
		// callers must also consult deny — we return non-nil list only for allow-only sets.
		// For deny-subset of *, return nil and let MaskRecord use DeniedFields helper.
		return nil
	}
	var out []string
	for f, eff := range fieldEffect {
		if eff == EffectAllow {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

// DeniedFields returns fields explicitly denied when * is allowed (override case).
func DeniedFields(entries []Entry, resource, action string, p PrincipalContext, record map[string]any) []string {
	matched := filterMatch(entries, resource, action)
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].Priority > matched[j].Priority
	})
	fieldEffect := map[string]string{}
	starEffect := ""
	for _, e := range matched {
		if record != nil && len(e.Domain) > 0 && !e.Domain.Match(record, p) {
			continue
		}
		if record == nil && len(e.Domain) > 0 {
			continue
		}
		fields := e.Fields
		if len(fields) == 0 {
			fields = []string{FieldsAll}
		}
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if f == FieldsAll {
				if starEffect == "" {
					starEffect = e.Effect
				}
				continue
			}
			if _, ok := fieldEffect[f]; !ok {
				fieldEffect[f] = e.Effect
			}
		}
	}
	if starEffect != EffectAllow {
		return nil
	}
	var denies []string
	for f, eff := range fieldEffect {
		if eff == EffectDeny {
			denies = append(denies, f)
		}
	}
	sort.Strings(denies)
	return denies
}

// BuildListFilter derives ListFilter from matching entries for the principal's roles.
func BuildListFilter(entries []Entry, resource, action string, p PrincipalContext) ListFilter {
	matched := filterMatch(entries, resource, action)
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].Priority > matched[j].Priority
	})

	var allows, denies []Domain
	unrestrictedAllow := false
	hasAllow := false

	// Track whether * / empty-domain allow survived denies at higher priority.
	type mark struct {
		effect string
		domain Domain
		prio   int
	}
	var applied []mark

	for _, e := range matched {
		fields := e.Fields
		if len(fields) == 0 {
			fields = []string{FieldsAll}
		}
		// List filter only cares about entries that affect the whole record (include *).
		coversAll := false
		for _, f := range fields {
			if f == FieldsAll || f == "" {
				coversAll = true
				break
			}
		}
		if !coversAll {
			continue
		}
		applied = append(applied, mark{effect: e.Effect, domain: e.Domain, prio: e.Priority})
	}

	if len(applied) == 0 {
		// field-only grants: treat as unrestricted list if any allow exists for action
		for _, e := range matched {
			if e.Effect == EffectAllow {
				return ListFilter{Unrestricted: true}
			}
		}
		return ListFilter{DenyAll: true}
	}

	for _, m := range applied {
		if m.effect == EffectAllow {
			hasAllow = true
			if len(m.domain) == 0 {
				unrestrictedAllow = true
			} else {
				allows = append(allows, m.domain)
			}
		} else if m.effect == EffectDeny {
			if len(m.domain) == 0 {
				// deny all records at this priority — unless lower allows exist,
				// empty deny means DenyAll if it's the winning model-level deny.
				// Higher-priority empty deny blocks everything.
				return ListFilter{DenyAll: true}
			}
			denies = append(denies, m.domain)
		}
	}

	if !hasAllow {
		return ListFilter{DenyAll: true}
	}
	if unrestrictedAllow && len(denies) == 0 {
		return ListFilter{Unrestricted: true}
	}
	return ListFilter{Allow: allows, Deny: denies, Unrestricted: unrestrictedAllow && len(denies) == 0}
}

func filterMatch(entries []Entry, resource, action string) []Entry {
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if !e.Active {
			continue
		}
		if e.Effect != EffectAllow && e.Effect != EffectDeny {
			continue
		}
		if !MatchResource(e.Resource, resource) {
			continue
		}
		if !MatchAction(e.Actions, action) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// MaskRecord removes forbidden keys from a record for read.
// allowed nil = all except denied; allowed non-nil = only those keys (+ system fields).
func MaskRecord(rec map[string]any, allowed []string, denied []string) map[string]any {
	if rec == nil {
		return nil
	}
	system := map[string]struct{}{
		"id": {}, "orgId": {}, "authorId": {}, "deleted": {}, "createdAt": {}, "updatedAt": {},
	}
	denySet := map[string]struct{}{}
	for _, d := range denied {
		denySet[d] = struct{}{}
	}
	out := map[string]any{}
	if allowed == nil {
		for k, v := range rec {
			if _, bad := denySet[k]; bad {
				continue
			}
			out[k] = v
		}
		return out
	}
	allowSet := map[string]struct{}{}
	for _, a := range allowed {
		allowSet[a] = struct{}{}
	}
	for k, v := range rec {
		if _, ok := system[k]; ok {
			out[k] = v
			continue
		}
		if _, bad := denySet[k]; bad {
			continue
		}
		if _, ok := allowSet[k]; ok {
			out[k] = v
		}
	}
	return out
}

// FilterWriteFields drops keys not allowed for write; returns error fields if any forbidden present.
func FilterWriteFields(fields map[string]any, allowed []string, denied []string) (clean map[string]any, forbidden []string) {
	denySet := map[string]struct{}{}
	for _, d := range denied {
		denySet[d] = struct{}{}
	}
	clean = map[string]any{}
	if allowed == nil {
		for k, v := range fields {
			if _, bad := denySet[k]; bad {
				forbidden = append(forbidden, k)
				continue
			}
			clean[k] = v
		}
		return clean, forbidden
	}
	allowSet := map[string]struct{}{}
	for _, a := range allowed {
		allowSet[a] = struct{}{}
	}
	for k, v := range fields {
		if _, bad := denySet[k]; bad {
			forbidden = append(forbidden, k)
			continue
		}
		if _, ok := allowSet[k]; !ok {
			forbidden = append(forbidden, k)
			continue
		}
		clean[k] = v
	}
	return clean, forbidden
}
