package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimitExcept applies a per-IP request limit to everything except the given
// path prefixes. Bulk transfer endpoints are excluded because a single large
// upload legitimately issues hundreds of sequential requests.
func RateLimitExcept(requests int, window time.Duration, prefixes ...string) func(http.Handler) http.Handler {
	limiter := httprate.LimitByIP(requests, window)
	return func(next http.Handler) http.Handler {
		limited := limiter(next)
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			for _, prefix := range prefixes {
				if strings.HasPrefix(request.URL.Path, prefix) {
					next.ServeHTTP(w, request)
					return
				}
			}
			limited.ServeHTTP(w, request)
		})
	}
}

// RateLimitPaths applies a stricter per-IP limit to exactly the given paths,
// bounding credential brute force against the authentication endpoints.
func RateLimitPaths(requests int, window time.Duration, paths ...string) func(http.Handler) http.Handler {
	limiter := httprate.LimitByIP(requests, window)
	return func(next http.Handler) http.Handler {
		limited := limiter(next)
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			for _, path := range paths {
				if request.URL.Path == path {
					limited.ServeHTTP(w, request)
					return
				}
			}
			next.ServeHTTP(w, request)
		})
	}
}
