package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseFileIndices(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/download?files=0,2,2,9&i=1", nil)
	got := parseFileIndices(r, 4)
	if len(got) != 3 || got[0] != 0 || got[1] != 2 || got[2] != 1 {
		t.Fatalf("%v", got)
	}
	r2 := httptest.NewRequest(http.MethodGet, "/download", nil)
	if parseFileIndices(r2, 4) != nil {
		t.Fatal("want nil for all")
	}
}
