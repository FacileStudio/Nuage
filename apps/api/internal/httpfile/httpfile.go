// Package httpfile writes stored file bodies as HTTP download responses.
package httpfile

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

// Serve answers with the file body through http.ServeContent, which is what
// makes the endpoint honour Range requests. Viewers that fetch pieces of a file
// depend on it: pdf.js reads the cross-reference table before any page, and in a
// file that is not linearized that table sits at the end, so a server that can
// only reply 200 with the whole body forces a full download before the first
// page can render.
//
// The modification time is deliberately zero. A file record's UpdatedAt moves on
// a rename, which does not change a byte, so it is a misleading If-Range
// validator; the content hash is the honest one and goes out as the ETag.
// ServeContent owns Content-Length and Content-Range, so neither is set here.
//
// The size is probed before any header is written. ServeContent seeks to learn
// the length and answers a failure itself, in plain text and outside the JSON
// error envelope every other endpoint uses; a storage key whose object is
// missing reaches exactly that path, because the object store opens lazily and
// only fails on the first seek. Probing first keeps the failure in the envelope
// and leaves the response unwritten, so the caller can still answer with it.
func Serve(w http.ResponseWriter, r *http.Request, record *schemas.File, content io.ReadSeeker) error {
	if _, err := content.Seek(0, io.SeekEnd); err != nil {
		return errors.Internal("failed to size stored object", err)
	}
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return errors.Internal("failed to rewind stored object", err)
	}

	w.Header().Set("Content-Type", record.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(record.Name)))
	if record.Hash != "" {
		w.Header().Set("ETag", `"`+record.Hash+`"`)
	}
	http.ServeContent(w, r, record.Name, time.Time{}, content)
	return nil
}
