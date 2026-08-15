package module

import (
	"strconv"
	"strings"
)

// VersionLess reports whether a is a lower dotted version than b (e.g. 0.1.0 < 0.2.0).
func VersionLess(a, b string) bool {
	as := versionParts(a)
	bs := versionParts(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av != bv {
			return av < bv
		}
	}
	return false
}

func versionParts(v string) []int {
	v = strings.TrimSpace(v)
	if v == "" {
		return []int{0}
	}
	chunks := strings.Split(v, ".")
	out := make([]int, 0, len(chunks))
	for _, c := range chunks {
		n, err := strconv.Atoi(c)
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
