package engine

import "testing"

func TestSlicePage(t *testing.T) {
	items := []Record{{"id": "1"}, {"id": "2"}, {"id": "3"}, {"id": "4"}, {"id": "5"}}

	full := slicePage(items, ListPageOpts{})
	if full.Total != 5 || len(full.Items) != 5 {
		t.Fatalf("full: got total=%d len=%d", full.Total, len(full.Items))
	}

	p1 := slicePage(items, ListPageOpts{Page: 1, PageSize: 2})
	if p1.Total != 5 || len(p1.Items) != 2 || p1.Items[0]["id"] != "1" {
		t.Fatalf("page1: %+v", p1)
	}

	p3 := slicePage(items, ListPageOpts{Page: 3, PageSize: 2})
	if p3.Total != 5 || len(p3.Items) != 1 || p3.Items[0]["id"] != "5" {
		t.Fatalf("page3: %+v", p3)
	}

	empty := slicePage(items, ListPageOpts{Page: 9, PageSize: 2})
	if empty.Total != 5 || len(empty.Items) != 0 {
		t.Fatalf("empty page: %+v", empty)
	}
}

func TestParseListPageOpts(t *testing.T) {
	opts := parseListPageOpts(map[string]any{"page": 2, "pageSize": 25})
	if opts.Page != 2 || opts.PageSize != 25 {
		t.Fatalf("got %+v", opts)
	}
	opts = parseListPageOpts(map[string]any{"page": float64(3), "pageSize": int32(10)})
	if opts.Page != 3 || opts.PageSize != 10 {
		t.Fatalf("coerced got %+v", opts)
	}
	opts = parseListPageOpts(nil)
	if opts.Page != 1 || opts.PageSize != 0 {
		t.Fatalf("defaults got %+v", opts)
	}
}
