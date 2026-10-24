package shell

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxCell = 48

// formatValue renders lists of records as a table, records as key/value lines,
// and scalars as-is. columns limits/orders record keys when set.
func formatValue(v any, columns []string) string {
	switch t := v.(type) {
	case nil:
		return "(null)"
	case []any:
		if len(t) == 0 {
			return "(no rows)"
		}
		if _, ok := t[0].(map[string]any); !ok {
			parts := make([]string, len(t))
			for i, item := range t {
				parts[i] = cell(item)
			}
			return strings.Join(parts, "\n")
		}
		cols := columns
		if len(cols) == 0 {
			cols = sortedKeys(t[0].(map[string]any))
		}
		rows := make([][]string, 0, len(t))
		for _, item := range t {
			rec, _ := item.(map[string]any)
			row := make([]string, len(cols))
			for i, c := range cols {
				row[i] = cell(rec[c])
			}
			rows = append(rows, row)
		}
		noun := "rows"
		if len(t) == 1 {
			noun = "row"
		}
		return table(cols, rows) + fmt.Sprintf("\n(%d %s)", len(t), noun)
	case map[string]any:
		cols := columns
		if len(cols) == 0 {
			cols = sortedKeys(t)
		}
		width := 0
		for _, c := range cols {
			if n := utf8.RuneCountInString(c); n > width {
				width = n
			}
		}
		lines := make([]string, 0, len(cols))
		for _, c := range cols {
			lines = append(lines, fmt.Sprintf("%-*s  %s", width, c, cell(t[c])))
		}
		return strings.Join(lines, "\n")
	default:
		return cell(t)
	}
}

func table(header []string, rows [][]string) string {
	widths := make([]int, len(header))
	for i, h := range header {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if n := utf8.RuneCountInString(c); i < len(widths) && n > widths[i] {
				widths[i] = n
			}
		}
	}
	var b strings.Builder
	line := func(cells []string) {
		for i, c := range cells {
			if i > 0 {
				b.WriteString("  ")
			}
			if i == len(cells)-1 {
				b.WriteString(c)
			} else {
				fmt.Fprintf(&b, "%-*s", widths[i], c)
			}
		}
		b.WriteString("\n")
	}
	line(header)
	sep := make([]string, len(header))
	for i, w := range widths {
		sep[i] = strings.Repeat("─", w)
	}
	line(sep)
	for _, r := range rows {
		line(r)
	}
	return strings.TrimRight(b.String(), "\n")
}

func cell(v any) string {
	var s string
	switch t := v.(type) {
	case nil:
		s = ""
	case string:
		s = t
	case []any:
		parts := make([]string, len(t))
		for i, item := range t {
			parts[i] = fmt.Sprint(item)
		}
		s = strings.Join(parts, ", ")
	case map[string]any:
		b, _ := json.Marshal(t)
		s = string(b)
	default:
		s = fmt.Sprint(t)
	}
	s = strings.ReplaceAll(s, "\n", " ")
	if utf8.RuneCountInString(s) > maxCell {
		r := []rune(s)
		s = string(r[:maxCell-1]) + "…"
	}
	return s
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func prettyJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
