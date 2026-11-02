package server

import (
	"net/http"
	"strconv"
	"strings"
)

// parseFileIndices reads ?files=0,2,3 and/or repeated ?i=0&i=2.
// Empty / missing means "all" (nil). Invalid indexes are skipped.
func parseFileIndices(r *http.Request, max int) []int {
	var raw []string
	if v := r.URL.Query().Get("files"); v != "" {
		raw = append(raw, strings.Split(v, ",")...)
	}
	raw = append(raw, r.URL.Query()["i"]...)
	if len(raw) == 0 {
		return nil
	}
	seen := map[int]bool{}
	var out []int
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n >= max || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}
