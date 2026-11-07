package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/httprate"
)

// RateLimitAPI applies a per-IP request limit to paths under prefix, except
// those exempt accepts. Everything outside the prefix, such as the client's
// static assets, is never counted: a cold page load alone fetches dozens of
// chunks and fonts and would otherwise eat the budget before the first call.
func RateLimitAPI(requests int, window time.Duration, prefix string, exempt func(path string) bool) func(http.Handler) http.Handler {
	limiter := httprate.LimitByIP(requests, window)
	return func(next http.Handler) http.Handler {
		limited := limiter(next)
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			path := request.URL.Path
			if !strings.HasPrefix(path, prefix) || exempt(path) {
				next.ServeHTTP(w, request)
				return
			}
			limited.ServeHTTP(w, request)
		})
	}
}

// IsTransferPath reports whether path moves file bytes: chunked uploads,
// downloads, thumbnails and presigned links. One large upload issues hundreds
// of chunk requests, pdf.js reads a document as many Range requests, and a grid
// of photos asks for one thumbnail per tile, so counting them against the
// request budget throttles ordinary use, not abuse.
func IsTransferPath(path string) bool {
	return strings.HasPrefix(path, "/api/files/upload/") ||
		strings.HasSuffix(path, "/download") ||
		strings.HasSuffix(path, "/thumbnail") ||
		strings.Contains(path, "/thumbnail/") ||
		strings.Contains(path, "/download/") ||
		strings.HasPrefix(path, "/api/presigned/")
}
