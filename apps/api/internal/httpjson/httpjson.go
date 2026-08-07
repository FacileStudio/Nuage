package httpjson

import (
	"encoding/json"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
)

const maxJSONBodyBytes int64 = 1 << 20

func DecodeJSON(w http.ResponseWriter, request *http.Request, dst any) error {
	defer request.Body.Close()
	request.Body = http.MaxBytesReader(w, request.Body, maxJSONBodyBytes)

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if stderrors.As(err, &maxBytesErr) {
			return errors.TooLarge("request body too large")
		}
		return errors.Invalid("invalid JSON body")
	}
	if err := decoder.Decode(new(struct{})); !stderrors.Is(err, io.EOF) {
		var maxBytesErr *http.MaxBytesError
		if stderrors.As(err, &maxBytesErr) {
			return errors.TooLarge("request body too large")
		}
		return errors.Invalid("request body must contain a single JSON object")
	}
	return nil
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, err error) {
	var appErr *errors.Error
	if !stderrors.As(err, &appErr) {
		appErr = errors.Internal("internal server error", err)
	}

	status := errors.Status(appErr)
	if status >= http.StatusInternalServerError {
		slog.Error("request failed",
			slog.String("code", appErr.Code),
			slog.String("message", appErr.Message),
			slog.Any("cause", stderrors.Unwrap(appErr)),
		)
	}

	WriteJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	})
}
