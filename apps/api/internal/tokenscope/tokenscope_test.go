package tokenscope

import (
	"net/http"
	"testing"
)

func TestPermitsKeepsThumbnailsAwayFromRestrictedTokens(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/api/files/12/download", true},
		{http.MethodGet, "/api/files/12/thumbnail", false},
		{http.MethodGet, "/files/12/thumbnail", false},
		{http.MethodPost, "/api/files/12/presign", true},
		{http.MethodGet, "/api/folders", false},
	}
	for _, c := range cases {
		if got := Permits(c.method, c.path); got != c.want {
			t.Errorf("Permits(%s, %q) = %v, want %v", c.method, c.path, got, c.want)
		}
	}
}
