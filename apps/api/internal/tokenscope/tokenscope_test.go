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
		{http.MethodPut, "/api/spaces/by-facile-id/fac_0123456789abcdef01234567", false},
	}
	for _, c := range cases {
		if got := Permits(c.method, c.path); got != c.want {
			t.Errorf("Permits(%s, %q) = %v, want %v", c.method, c.path, got, c.want)
		}
	}
}
