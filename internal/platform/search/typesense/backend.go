package typesense

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"kaizengo/internal/platform/search"
)

type backend struct {
	baseURL  string
	apiKey   string
	client   *http.Client
	fallback search.Backend

	ensuredMu sync.Mutex
	ensured   map[string]bool
}

func newFromEnv() search.Backend {
	url := strings.TrimSpace(os.Getenv("KaizenGo_TYPESENSE_URL"))
	if url == "" {
		return nil
	}
	key := strings.TrimSpace(os.Getenv("KaizenGo_TYPESENSE_API_KEY"))
	return &backend{
		baseURL:  strings.TrimRight(url, "/"),
		apiKey:   key,
		client:   &http.Client{Timeout: 10 * time.Second},
		fallback: newMemoryFallback(),
		ensured:  map[string]bool{},
	}
}

func newMemoryFallback() search.Backend {
	return search.NewMemoryBackend()
}

// RegisterFromEnv replaces the search backend when Typesense env is configured
// and installs query middleware so indexed-field searches hit Typesense first.
func RegisterFromEnv() {
	b := newFromEnv()
	if b == nil {
		return
	}
	search.Register(b)
	if tb, ok := b.(*backend); ok {
		search.UseQuery(tb.AsQueryMiddleware())
	}
}

func (b *backend) Upsert(ctx context.Context, doc search.Document) error {
	if err := b.fallback.Upsert(ctx, doc); err != nil {
		return err
	}
	if b.baseURL == "" {
		return nil
	}
	if err := b.ensureCollection(ctx, doc.Collection); err != nil {
		return err
	}
	payload := map[string]any{
		"id":         doc.ID,
		"org_id":     doc.OrgID,
		"collection": doc.Collection,
		"title":      doc.Title,
		"body":       doc.Body,
	}
	for k, v := range doc.Fields {
		payload[k] = v
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.baseURL+"/collections/"+doc.Collection+"/documents?action=upsert",
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	b.setAPIKey(req)
	res, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("typesense upsert: %s", strings.TrimSpace(string(raw)))
	}
	return nil
}

func (b *backend) CollectionCounts() map[string]int {
	if c, ok := b.fallback.(search.Counter); ok {
		return c.CollectionCounts()
	}
	return map[string]int{}
}

func (b *backend) Delete(ctx context.Context, collection, orgID, id string) error {
	if err := b.fallback.Delete(ctx, collection, orgID, id); err != nil {
		return err
	}
	if b.baseURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		b.baseURL+"/collections/"+collection+"/documents/"+id,
		nil,
	)
	if err != nil {
		return err
	}
	b.setAPIKey(req)
	res, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 && res.StatusCode != 404 {
		raw, _ := io.ReadAll(res.Body)
		return fmt.Errorf("typesense delete: %s", strings.TrimSpace(string(raw)))
	}
	return nil
}

func (b *backend) setAPIKey(req *http.Request) {
	if b.apiKey != "" {
		req.Header.Set("X-TYPESENSE-API-KEY", b.apiKey)
	}
}

// ensureCollection creates the Typesense collection if it does not exist yet.
func (b *backend) ensureCollection(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("typesense: empty collection name")
	}
	b.ensuredMu.Lock()
	defer b.ensuredMu.Unlock()
	if b.ensured[name] {
		return nil
	}
	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet,
		b.baseURL+"/collections/"+name, nil)
	if err != nil {
		return err
	}
	b.setAPIKey(getReq)
	getRes, err := b.client.Do(getReq)
	if err != nil {
		return err
	}
	defer getRes.Body.Close()
	if getRes.StatusCode == http.StatusOK {
		b.ensured[name] = true
		return nil
	}
	if getRes.StatusCode != http.StatusNotFound {
		raw, _ := io.ReadAll(getRes.Body)
		return fmt.Errorf("typesense get collection: %s", strings.TrimSpace(string(raw)))
	}

	schema := map[string]any{
		"name": name,
		"fields": []map[string]any{
			{"name": "org_id", "type": "string", "facet": true},
			{"name": "title", "type": "string"},
			{"name": "body", "type": "string", "optional": true},
			{"name": "collection", "type": "string", "facet": true, "optional": true},
			{"name": ".*", "type": "auto"},
		},
	}
	body, _ := json.Marshal(schema)
	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		b.baseURL+"/collections", bytes.NewReader(body))
	if err != nil {
		return err
	}
	createReq.Header.Set("Content-Type", "application/json")
	b.setAPIKey(createReq)
	createRes, err := b.client.Do(createReq)
	if err != nil {
		return err
	}
	defer createRes.Body.Close()
	if createRes.StatusCode >= 300 && createRes.StatusCode != http.StatusConflict {
		raw, _ := io.ReadAll(createRes.Body)
		return fmt.Errorf("typesense create collection: %s", strings.TrimSpace(string(raw)))
	}
	b.ensured[name] = true
	return nil
}
