package tests

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
