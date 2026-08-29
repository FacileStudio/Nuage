package httpfile

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

// TestServe pins the response contract the client depends on: the record's own
// MIME type and an attachment disposition on every reply, plus the range
// answers a PDF viewer needs to render a page without reading the whole object.
func TestServe(t *testing.T) {
	record := &schemas.File{
		Name:     "rapport final.pdf",
		MimeType: "application/pdf",
		Size:     10,
		Hash:     "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
	}
	const content = "0123456789"

	cases := []struct {
		name        string
		rangeHeader string
		wantStatus  int
		wantRange   string
		wantBody    string
	}{
		{name: "full body", wantStatus: http.StatusOK, wantBody: content},
		{name: "first bytes", rangeHeader: "bytes=0-3", wantStatus: http.StatusPartialContent, wantRange: "bytes 0-3/10", wantBody: "0123"},
		{name: "tail", rangeHeader: "bytes=-2", wantStatus: http.StatusPartialContent, wantRange: "bytes 8-9/10", wantBody: "89"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/files/1/download", nil)
			if tc.rangeHeader != "" {
				req.Header.Set("Range", tc.rangeHeader)
			}
			w := httptest.NewRecorder()

			Serve(w, req, record, strings.NewReader(content))

			resp := w.Result()
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			if got := resp.Header.Get("Accept-Ranges"); got != "bytes" {
				t.Errorf("Accept-Ranges = %q, want %q", got, "bytes")
			}
			if got := resp.Header.Get("Content-Type"); got != record.MimeType {
				t.Errorf("Content-Type = %q, want %q", got, record.MimeType)
			}
			want := "attachment; filename*=UTF-8''rapport%20final.pdf"
			if got := resp.Header.Get("Content-Disposition"); got != want {
				t.Errorf("Content-Disposition = %q, want %q", got, want)
			}
			if got := resp.Header.Get("Content-Range"); got != tc.wantRange {
				t.Errorf("Content-Range = %q, want %q", got, tc.wantRange)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if string(body) != tc.wantBody {
				t.Errorf("body = %q, want %q", body, tc.wantBody)
			}
		})
	}
}
