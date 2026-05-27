package docs

import (
	_ "embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed openapi.yaml
var specBytes []byte

func RegisterRoutes(router chi.Router) {
	router.Get("/docs", serveDocs)
	router.Get("/docs/openapi.yaml", serveSpec)
}

func serveSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(specBytes)
}

func serveDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(docsHTML))
}

const docsHTML = `<!doctype html>
<html>
<head>
  <title>Nuage API</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { margin: 0; }
  </style>
</head>
<body>
  <script id="api-reference" data-url="/docs/openapi.yaml"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`
