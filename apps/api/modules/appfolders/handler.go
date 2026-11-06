package appfolders

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"
	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpfile"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// ObjectResponse is one stored object as an app sees it.
type ObjectResponse struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ObjectListResponse is a page of objects, with the key to pass as after for
// the next page, empty on the last one.
type ObjectListResponse struct {
	Objects []ObjectResponse `json:"objects"`
	Next    string           `json:"next,omitempty"`
}

// AppDataResponse lists what each app stores for the user.
type AppDataResponse struct {
	Apps []AppData `json:"apps"`
}

type handler struct {
	service *Service
}

// caller is the user and, on the app routes, the registered app the request
// acts as. RequireAuth put both in the context.
func caller(r *http.Request) (int64, *schemas.RegisteredApp, error) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		return 0, nil, errors.Unauthorized("missing auth")
	}
	userID, err := strconv.ParseInt(identity.UserID, 10, 64)
	if err != nil {
		return 0, nil, errors.Internal("failed to parse user id", err)
	}
	return userID, appregistry.From(r.Context()), nil
}

// appCaller is caller on a route only a registered app may use.
func appCaller(r *http.Request) (int64, *schemas.RegisteredApp, error) {
	userID, app, err := caller(r)
	if err == nil && app == nil {
		err = errors.Forbidden("only a registered app acting for a user can use app folders")
	}
	return userID, app, err
}

func toResponse(object schemas.AppObject) ObjectResponse {
	return ObjectResponse{
		Key: object.Key, Name: object.Name, MimeType: object.MimeType, Size: object.Size, Hash: object.Hash,
		CreatedAt: object.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: object.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func (h *handler) put(w http.ResponseWriter, r *http.Request) {
	userID, app, err := appCaller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if r.ContentLength > MaxObjectSize {
		httpjson.WriteError(w, errors.TooLarge("object is too large"))
		return
	}
	query := r.URL.Query()
	object, created, err := h.service.Put(r.Context(), userID, app, Upload{
		Key: query.Get("key"), Name: query.Get("name"), MimeType: r.Header.Get("Content-Type"),
		Size: r.ContentLength, Body: r.Body,
	})
	if err != nil {
		entitlements.WriteError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpjson.WriteJSON(w, status, toResponse(*object))
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	userID, app, err := appCaller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	object, err := h.service.Find(r.Context(), userID, app.ID, r.URL.Query().Get("key"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	content, err := h.service.Open(r.Context(), object)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	defer content.Close()
	record := &schemas.File{Name: object.Name, MimeType: object.MimeType, Hash: object.Hash}
	if err := httpfile.Serve(w, r, record, content); err != nil {
		httpjson.WriteError(w, err)
	}
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	userID, app, err := appCaller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > maxListLimit {
		limit = defaultListLimit
	}
	objects, err := h.service.List(r.Context(), userID, app.ID, ListQuery{Prefix: query.Get("prefix"), After: query.Get("after"), Limit: limit + 1})
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	response := ObjectListResponse{Objects: make([]ObjectResponse, 0, len(objects))}
	if len(objects) > limit {
		objects = objects[:limit]
		response.Next = objects[limit-1].Key
	}
	for _, object := range objects {
		response.Objects = append(response.Objects, toResponse(object))
	}
	httpjson.WriteJSON(w, http.StatusOK, response)
}

func (h *handler) remove(w http.ResponseWriter, r *http.Request) {
	userID, app, err := appCaller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := h.service.Delete(r.Context(), userID, app.ID, r.URL.Query().Get("key")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
