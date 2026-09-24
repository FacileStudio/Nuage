package tests

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func davRequest(ts *testServer, method, path, token string, body string) *http.Response {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.SetBasicAuth("user@example.com", token)
	if method == "PROPFIND" {
		req.Header.Set("Depth", "1")
		req.Header.Set("Content-Type", "application/xml")
	}
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	return w.Result()
}

func TestWebDAVOptionsRequiresAuth(t *testing.T) {
	ts := setupTestServer(t)

	req := httptest.NewRequest("OPTIONS", "/webdav/", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("DAV"), "1, 2")
	assert.NotEmpty(t, resp.Header.Get("WWW-Authenticate"))
}

func TestWebDAVPropfindRoot(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav@example.com", "password12345")

	resp := davRequest(ts, "PROPFIND", "/webdav/", token, "")
	assert.Equal(t, 207, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "multistatus")
}

func TestWebDAVMkcolAndList(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-mkdir@example.com", "password12345")

	resp := davRequest(ts, "MKCOL", "/webdav/TestFolder", token, "")
	require.Equal(t, 201, resp.StatusCode)

	resp = davRequest(ts, "PROPFIND", "/webdav/", token, "")
	require.Equal(t, 207, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "TestFolder")
}

func TestWebDAVPutAndGet(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-put@example.com", "password12345")

	content := "hello webdav world"
	req := httptest.NewRequest("PUT", "/webdav/hello.txt", strings.NewReader(content))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, 201, resp.StatusCode)

	resp = davRequest(ts, "GET", "/webdav/hello.txt", token, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, content, string(body))
}

func TestWebDAVDelete(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-del@example.com", "password12345")

	req := httptest.NewRequest("PUT", "/webdav/todelete.txt", strings.NewReader("data"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.Equal(t, 201, w.Result().StatusCode)

	resp := davRequest(ts, "DELETE", "/webdav/todelete.txt", token, "")
	assert.Equal(t, 204, resp.StatusCode)

	resp = davRequest(ts, "GET", "/webdav/todelete.txt", token, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWebDAVMove(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-mv@example.com", "password12345")

	req := httptest.NewRequest("PUT", "/webdav/original.txt", strings.NewReader("move me"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.Equal(t, 201, w.Result().StatusCode)

	req = httptest.NewRequest("MOVE", "/webdav/original.txt", nil)
	req.SetBasicAuth("user@example.com", token)
	req.Header.Set("Destination", "/webdav/renamed.txt")
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	resp := w.Result()
	assert.True(t, resp.StatusCode == 201 || resp.StatusCode == 204,
		fmt.Sprintf("expected 201 or 204, got %d", resp.StatusCode))

	resp = davRequest(ts, "GET", "/webdav/renamed.txt", token, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "move me", string(body))
}

func TestWebDAVPutInFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-putfolder@example.com", "password12345")

	resp := davRequest(ts, "MKCOL", "/webdav/Docs", token, "")
	require.Equal(t, 201, resp.StatusCode)

	req := httptest.NewRequest("PUT", "/webdav/Docs/file.txt", strings.NewReader("in folder"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.Equal(t, 201, w.Result().StatusCode)

	resp = davRequest(ts, "PROPFIND", "/webdav/Docs", token, "")
	require.Equal(t, 207, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "file.txt")
}

func TestWebDAVOverwrite(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-overwrite@example.com", "password12345")

	req := httptest.NewRequest("PUT", "/webdav/update.txt", strings.NewReader("version1"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.Equal(t, 201, w.Result().StatusCode)

	req = httptest.NewRequest("PUT", "/webdav/update.txt", strings.NewReader("version2"))
	req.SetBasicAuth("user@example.com", token)
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.True(t, w.Result().StatusCode == 200 || w.Result().StatusCode == 204 || w.Result().StatusCode == 201)

	resp := davRequest(ts, "GET", "/webdav/update.txt", token, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "version2", string(body))
}

func TestWebDAVDeleteFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-delfolder@example.com", "password12345")

	davRequest(ts, "MKCOL", "/webdav/ToDelete", token, "")

	req := httptest.NewRequest("PUT", "/webdav/ToDelete/inner.txt", strings.NewReader("content"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	resp := davRequest(ts, "DELETE", "/webdav/ToDelete", token, "")
	assert.Equal(t, 204, resp.StatusCode)

	resp = davRequest(ts, "PROPFIND", "/webdav/ToDelete", token, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWebDAVQuotaEnforced(t *testing.T) {
	ts := setupTestServer(t)
	userID, token := registerUser(ts, "dav-quota@example.com", "password12345")
	uid, err := strconv.ParseInt(userID, 10, 64)
	require.NoError(t, err)

	q := schemas.UserQuota{UserID: uid}
	require.NoError(t, ts.db.Where(schemas.UserQuota{UserID: uid}).
		Assign(schemas.UserQuota{StorageLimit: 10}).FirstOrCreate(&q).Error)

	req := httptest.NewRequest("PUT", "/webdav/big.txt", strings.NewReader(strings.Repeat("x", 100)))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	assert.GreaterOrEqual(t, w.Result().StatusCode, 400)

	resp := davRequest(ts, "GET", "/webdav/big.txt", token, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWebDAVQuotaUsageCharged(t *testing.T) {
	ts := setupTestServer(t)
	userID, token := registerUser(ts, "dav-usage@example.com", "password12345")
	uid, err := strconv.ParseInt(userID, 10, 64)
	require.NoError(t, err)

	content := "twelve bytes"
	req := httptest.NewRequest("PUT", "/webdav/counted.txt", strings.NewReader(content))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.Equal(t, 201, w.Result().StatusCode)

	var q schemas.UserQuota
	require.NoError(t, ts.db.Where("user_id = ?", uid).First(&q).Error)
	assert.Equal(t, int64(len(content)), q.StorageUsed)
}

func TestWebDAVJunkNameDoesNotShadowFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-junk@example.com", "password12345")

	resp := davRequest(ts, "MKCOL", "/webdav/._archive", token, "")
	require.Equal(t, 201, resp.StatusCode)

	req := httptest.NewRequest("PUT", "/webdav/._archive/real.txt", strings.NewReader("keep me"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.Equal(t, 201, w.Result().StatusCode)

	resp = davRequest(ts, "GET", "/webdav/._archive/real.txt", token, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "keep me", string(body))
}

func TestWebDAVRangeGet(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-range@example.com", "password12345")

	req := httptest.NewRequest("PUT", "/webdav/range.txt", strings.NewReader("0123456789"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.Equal(t, 201, w.Result().StatusCode)

	req = httptest.NewRequest("GET", "/webdav/range.txt", nil)
	req.SetBasicAuth("user@example.com", token)
	req.Header.Set("Range", "bytes=2-5")
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusPartialContent, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "2345", string(body))
}

func TestWebDAVDSStoreIgnored(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-ds@example.com", "password12345")

	req := httptest.NewRequest("PUT", "/webdav/.DS_Store", strings.NewReader("junk"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)

	resp := davRequest(ts, "PROPFIND", "/webdav/", token, "")
	body, _ := io.ReadAll(resp.Body)
	assert.NotContains(t, string(body), ".DS_Store")
}

func davSpaceBase(id int64) string {
	return fmt.Sprintf("/webdav/spaces/%d/", id)
}

func TestWebDAVSpacesPropfindListsMemberSpaces(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "dav-space-list@example.com", "password12345")
	_, foreign := registerUser(ts, "dav-space-list-other@example.com", "password12345")

	memberSpace := createSpace(t, ts, owner, "Member Space")
	foreignSpace := createSpace(t, ts, foreign, "Foreign Space")

	resp := davRequest(ts, "PROPFIND", "/webdav/spaces/", owner, "")
	require.Equal(t, 207, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), fmt.Sprintf("/webdav/spaces/%d", memberSpace))
	assert.NotContains(t, string(body), fmt.Sprintf("/webdav/spaces/%d", foreignSpace))
}

func TestWebDAVSpacesPropfindMemberMount(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-space-mount@example.com", "password12345")
	spaceID := createSpace(t, ts, token, "Mount Space")

	resp := davRequest(ts, "PROPFIND", davSpaceBase(spaceID), token, "")
	assert.Equal(t, 207, resp.StatusCode)
}

func TestWebDAVSpacesPropfindNonMemberIsNotFound(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "dav-space-closed@example.com", "password12345")
	_, outsider := registerUser(ts, "dav-space-outsider@example.com", "password12345")
	spaceID := createSpace(t, ts, owner, "Closed Space")

	resp := davRequest(ts, "PROPFIND", davSpaceBase(spaceID), outsider, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resp.StatusCode)
}

func TestWebDAVSpacesNonNumericSegmentIsNotFound(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-space-junkid@example.com", "password12345")

	resp := davRequest(ts, "PROPFIND", "/webdav/spaces/abc/", token, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWebDAVSpacesPutStampsSpaceID(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-space-put@example.com", "password12345")
	spaceID := createSpace(t, ts, token, "Upload Space")

	resp := davRequest(ts, "PUT", davSpaceBase(spaceID)+"note.txt", token, "space note")
	require.Equal(t, 201, resp.StatusCode)

	getResp := davRequest(ts, "GET", davSpaceBase(spaceID)+"note.txt", token, "")
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	body, _ := io.ReadAll(getResp.Body)
	assert.Equal(t, "space note", string(body))

	var file schemas.File
	require.NoError(t, ts.db.Where("name = ?", "note.txt").First(&file).Error)
	require.NotNil(t, file.SpaceID, "a space mount upload must stamp space_id")
	assert.Equal(t, spaceID, *file.SpaceID)
}

func TestWebDAVSpacesMkcolStampsSpaceID(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-space-mkcol@example.com", "password12345")
	spaceID := createSpace(t, ts, token, "Folder Space")

	resp := davRequest(ts, "MKCOL", davSpaceBase(spaceID)+"sub", token, "")
	require.Equal(t, 201, resp.StatusCode)

	var folder schemas.Folder
	require.NoError(t, ts.db.Where("name = ?", "sub").First(&folder).Error)
	require.NotNil(t, folder.SpaceID, "a space mount mkcol must stamp space_id")
	assert.Equal(t, spaceID, *folder.SpaceID)
}

func TestWebDAVSpacesAnyMemberCanDelete(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "dav-space-ownerdel@example.com", "password12345")
	memberID, member := registerUser(ts, "dav-space-memberdel@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Delete Space")
	addSpaceMember(t, ts, owner, spaceID, memberID)
	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, owner, "owned.bin", "owned", spaceID).StatusCode)

	resp := davRequest(ts, "DELETE", davSpaceBase(spaceID)+"owned.bin", member, "")
	assert.Equal(t, 204, resp.StatusCode)

	after := davRequest(ts, "GET", davSpaceBase(spaceID)+"owned.bin", member, "")
	assert.Equal(t, http.StatusNotFound, after.StatusCode)
}

func TestWebDAVPersonalRootExcludesSpaceFolders(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-space-leak@example.com", "password12345")
	spaceID := createSpace(t, ts, token, "Leak Space")

	folderResp := doJSON(ts, "POST", "/folders", map[string]any{"name": "spacefoldersonly", "space_id": spaceID}, token)
	require.Equal(t, http.StatusCreated, folderResp.StatusCode)
	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, token, "spacefileonly.txt", "hidden", spaceID).StatusCode)

	resp := davRequest(ts, "PROPFIND", "/webdav/", token, "")
	require.Equal(t, 207, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.NotContains(t, string(body), "spacefoldersonly")
	assert.NotContains(t, string(body), "spacefileonly.txt")
}

func TestWebDAVSpacesMoveAcrossMountsFails(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-space-move@example.com", "password12345")
	spaceA := createSpace(t, ts, token, "Move From")
	spaceB := createSpace(t, ts, token, "Move To")
	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, token, "x.txt", "cross mount", spaceA).StatusCode)

	req := httptest.NewRequest("MOVE", davSpaceBase(spaceA)+"x.txt", nil)
	req.SetBasicAuth("user@example.com", token)
	req.Header.Set("Destination", davSpaceBase(spaceB)+"x.txt")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadGateway, w.Result().StatusCode)

	source := davRequest(ts, "GET", davSpaceBase(spaceA)+"x.txt", token, "")
	require.Equal(t, http.StatusOK, source.StatusCode)
	body, _ := io.ReadAll(source.Body)
	assert.Equal(t, "cross mount", string(body))

	assert.Equal(t, http.StatusNotFound, davRequest(ts, "GET", davSpaceBase(spaceB)+"x.txt", token, "").StatusCode)

	var moved int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("space_id = ?", spaceB).Count(&moved).Error)
	assert.Zero(t, moved, "a refused cross-mount move must create nothing in the destination space")
}

func TestWebDAVSpacesOptionsRequiresAuth(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-space-options@example.com", "password12345")
	spaceID := createSpace(t, ts, token, "Options Space")

	req := httptest.NewRequest("OPTIONS", davSpaceBase(spaceID), nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	resp := w.Result()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("WWW-Authenticate"))
}

func TestWebDAVSpacesRedirectsToTrailingSlash(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-space-redirect@example.com", "password12345")
	spaceID := createSpace(t, ts, token, "Redirect Space")

	req := httptest.NewRequest("PROPFIND", fmt.Sprintf("/webdav/spaces/%d", spaceID), nil)
	req.SetBasicAuth("user@example.com", token)
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Location"), davSpaceBase(spaceID))
}

func TestWebDAVSpacesIndexRedirectsToTrailingSlash(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-index-redirect@example.com", "password12345")

	resp := davRequest(ts, "PROPFIND", "/webdav/spaces", token, "")
	require.Equal(t, http.StatusMovedPermanently, resp.StatusCode)
	assert.Equal(t, "/webdav/spaces/", resp.Header.Get("Location"))
}

func TestWebDAVSpacesIndexRefusesWrites(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-index-ro@example.com", "password12345")

	req := httptest.NewRequest("PUT", "/webdav/spaces/", strings.NewReader(""))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.GreaterOrEqual(t, w.Result().StatusCode, 400)

	resp := davRequest(ts, "MKCOL", "/webdav/spaces/newspace", token, "")
	require.GreaterOrEqual(t, resp.StatusCode, 400)

	resp = davRequest(ts, "DELETE", "/webdav/spaces/", token, "")
	require.GreaterOrEqual(t, resp.StatusCode, 400)

	var folders int64
	require.NoError(t, ts.db.Model(&schemas.Folder{}).Where("name = ?", "newspace").Count(&folders).Error)
	assert.Zero(t, folders)

	var files int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Count(&files).Error)
	assert.Zero(t, files)
}

func TestWebDAVSpacesMoveWithinSpaceSucceeds(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-move-within@example.com", "password12345")
	spaceID := createSpace(t, ts, token, "Move Within")
	base := davSpaceBase(spaceID)

	req := httptest.NewRequest("PUT", base+"original.txt", strings.NewReader("move me"))
	req.SetBasicAuth("user@example.com", token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Result().StatusCode)

	req = httptest.NewRequest("MOVE", base+"original.txt", nil)
	req.SetBasicAuth("user@example.com", token)
	req.Header.Set("Destination", base+"renamed.txt")
	w = httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.True(t, w.Result().StatusCode == 201 || w.Result().StatusCode == 204,
		fmt.Sprintf("expected 201 or 204, got %d", w.Result().StatusCode))

	resp := davRequest(ts, "GET", base+"renamed.txt", token, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "move me", string(body))

	var record schemas.File
	require.NoError(t, ts.db.Where("name = ?", "renamed.txt").First(&record).Error)
	require.NotNil(t, record.SpaceID)
	assert.Equal(t, spaceID, *record.SpaceID)
}
