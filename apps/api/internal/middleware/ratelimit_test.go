package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsTransferPathCoversByteTransfersOnly(t *testing.T) {
	cases := map[string]bool{
		"/api/files/12/download":              true,
		"/api/files/12/thumbnail":             true,
		"/api/shared/tok/thumbnail/9":         true,
		"/api/shared/tok/download/9":          true,
		"/api/files/upload/abc/part/3":        true,
		"/api/presigned/token":                true,
		"/api/files":                          false,
		"/api/files/12":                       false,
		"/api/folders":                        false,
		"/api/auth/login":                     false,
		"/api/files/12/versions":              false,
		"/api/files/12/thumbnails-are-nice":   false,
		"/api/files/thumbnail-folder/12/link": false,
	}
	for path, want := range cases {
		if got := IsTransferPath(path); got != want {
			t.Errorf("IsTransferPath(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestRateLimitAPICountsOneIPv6PrefixAndSkipsExemptPaths(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := RateLimitAPI(2, time.Minute, "/api/", IsTransferPath)(ok)
	send := func(path, addr string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.RemoteAddr = addr
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		return recorder
	}
	for _, addr := range []string{"[2001:db8::1]:1000", "[2001:db8::2]:2000"} {
		if code := send("/api/files", addr).Code; code != http.StatusNoContent {
			t.Fatalf("request within the budget answered %d", code)
		}
	}
	refused := send("/api/files", "[2001:db8::3]:3000")
	if refused.Code != http.StatusTooManyRequests || refused.Header().Get("Retry-After") == "" {
		t.Fatalf("third request from one /64 answered %d with Retry-After %q", refused.Code, refused.Header().Get("Retry-After"))
	}
	if refused.Header().Get("X-RateLimit-Limit") != "" {
		t.Fatal("the limiter still sends X-RateLimit headers")
	}
	for _, path := range []string{"/api/files/9/download", "/app.js"} {
		if code := send(path, "[2001:db8::4]:4000").Code; code != http.StatusNoContent {
			t.Fatalf("%s was counted: %d", path, code)
		}
	}
	if code := send("/api/files", "198.51.100.7:5000").Code; code != http.StatusNoContent {
		t.Fatalf("another visitor shared the bucket: %d", code)
	}
}
