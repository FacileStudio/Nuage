package webdav

import (
	stderrors "errors"
	"io"
	"net/http"

	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"
)

// refuse remembers a storage refusal so the response says 507 Insufficient
// Storage instead of the 405 x/net/webdav answers for any failed write.
func (fs *NuageFS) refuse(err error) error {
	var apiErr *errors.Error
	if stderrors.Is(err, forfait.ErrLimit) || (stderrors.As(err, &apiErr) && errors.Status(apiErr) == http.StatusRequestEntityTooLarge) {
		fs.refusal = err
	}
	return err
}

// refusalWriter turns the error status x/net/webdav writes for a refused
// write into 507 Insufficient Storage, which WebDAV clients show as "disk
// full", and drops the body that went with the old status.
type refusalWriter struct {
	http.ResponseWriter
	fs        *NuageFS
	rewritten bool
}

// WriteHeader rewrites an error status when the write was refused for space.
func (w *refusalWriter) WriteHeader(status int) {
	if status >= http.StatusBadRequest && w.fs.refusal != nil {
		w.rewritten = true
		w.ResponseWriter.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.ResponseWriter.WriteHeader(http.StatusInsufficientStorage)
		_, _ = io.WriteString(w.ResponseWriter, http.StatusText(http.StatusInsufficientStorage)+"\n")
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

// Write drops the old status text once the status was rewritten.
func (w *refusalWriter) Write(b []byte) (int, error) {
	if w.rewritten {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
