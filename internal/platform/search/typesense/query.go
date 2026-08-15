package typesense

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"kaizengo/internal/platform/search"
)

// AsQueryMiddleware returns middleware that intercepts indexed-field search
// and serves Typesense results, falling through to next on error or when unset.
func (b *backend) AsQueryMiddleware() search.QueryMiddleware {
	return func(next search.QueryFunc) search.QueryFunc {
		return func(ctx context.Context, orgID, q string, collections []string, limit int) ([]search.Hit, error) {
			if b == nil || b.baseURL == "" {
				return next(ctx, orgID, q, collections, limit)
			}
			hits, err := b.queryTypesense(ctx, orgID, q, collections, limit)
			if err != nil {
				return next(ctx, orgID, q, collections, limit)
			}
			return hits, nil
		}
	}
}

func (b *backend) Query(ctx context.Context, orgID, q string, collections []string, limit int) ([]search.Hit, error) {
	// Writes dual-index into memory; Query is normally handled by AsQueryMiddleware.
	// Keep memory fallback when middleware is not registered.
	return b.fallback.Query(ctx, orgID, q, collections, limit)
}

func (b *backend) Name() string {
	return "typesense"
}

func (b *backend) queryTypesense(ctx context.Context, orgID, q string, collections []string, limit int) ([]search.Hit, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return []search.Hit{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	targets := collections
	if len(targets) == 0 {
		for name := range b.CollectionCounts() {
			targets = append(targets, name)
		}
	}
	if len(targets) == 0 {
		return []search.Hit{}, nil
	}

	out := make([]search.Hit, 0, limit)
	for _, collection := range targets {
		collection = strings.TrimSpace(collection)
		if collection == "" {
			continue
		}
		if err := b.ensureCollection(ctx, collection); err != nil {
			return nil, err
		}
		hits, err := b.searchCollection(ctx, collection, orgID, q, queryByForCollection(collection), limit)
		if err != nil {
			return nil, err
		}
		out = append(out, hits...)
		if len(out) >= limit {
			break
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func queryByForCollection(collection string) []string {
	parts := strings.SplitN(collection, ".", 2)
	fields := []string{"title", "body"}
	if len(parts) == 2 {
		if _, indexed, enabled := search.EffectiveIndex(parts[0], parts[1]); enabled && len(indexed) > 0 {
			fields = append([]string{}, indexed...)
		}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(fields)+2)
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	for _, extra := range []string{"title", "body"} {
		if _, ok := seen[extra]; !ok {
			out = append(out, extra)
			seen[extra] = struct{}{}
		}
	}
	return out
}

type tsSearchResponse struct {
	Hits []struct {
		Document map[string]any `json:"document"`
		TextMatch float64       `json:"text_match"`
	} `json:"hits"`
}

func (b *backend) searchCollection(
	ctx context.Context,
	collection, orgID, q string,
	queryBy []string,
	limit int,
) ([]search.Hit, error) {
	params := url.Values{}
	params.Set("q", q)
	params.Set("query_by", strings.Join(queryBy, ","))
	params.Set("filter_by", "org_id:="+orgID)
	params.Set("per_page", strconv.Itoa(limit))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		b.baseURL+"/collections/"+url.PathEscape(collection)+"/documents/search?"+params.Encode(),
		nil,
	)
	if err != nil {
		return nil, err
	}
	b.setAPIKey(req)
	res, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("typesense search: %s", strings.TrimSpace(string(raw)))
	}
	var parsed tsSearchResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := make([]search.Hit, 0, len(parsed.Hits))
	for _, h := range parsed.Hits {
		doc := h.Document
		id, _ := doc["id"].(string)
		if id == "" {
			continue
		}
		title, _ := doc["title"].(string)
		if title == "" {
			title = id
		}
		snippet := title
		if body, ok := doc["body"].(string); ok && body != "" {
			snippet = title + " — " + truncate(body, 120)
		}
		out = append(out, search.Hit{
			ID:         id,
			Collection: collection,
			Title:      title,
			Snippet:    snippet,
			Score:      h.TextMatch,
		})
	}
	return out, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
