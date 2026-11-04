package tests

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// davPut PUTs content at path over WebDAV.
func davPut(e planEnv, path, content string) int {
	req := httptest.NewRequest("PUT", path, strings.NewReader(content))
	req.SetBasicAuth("user@example.com", e.token)
	w := httptest.NewRecorder()
	e.ts.router.ServeHTTP(w, req)
	return w.Result().StatusCode
}

func TestPlanChunkedUploadChecksDeclaredAndFinalSize(t *testing.T) {
	e := newPlanEnv(t, "plan-chunked@example.com")
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-100)

	over := initChunkedUpload(e.ts, e.token, map[string]any{"file_name": "big.bin", "total_size": 200})
	assert.Equal(t, http.StatusPaymentRequired, over.StatusCode)

	sneaky := initSession(t, e.ts, e.token, 50)
	assert.Equal(t, http.StatusRequestEntityTooLarge, putChunk(e.ts, e.token, sneaky, 1, make([]byte, 150)).StatusCode)

	late := initSession(t, e.ts, e.token, 50)
	require.Equal(t, http.StatusCreated, putChunk(e.ts, e.token, late, 1, make([]byte, 50)).StatusCode)
	e.fill(t, nil, 60)
	assert.Equal(t, http.StatusPaymentRequired, completeChunked(e.ts, e.token, late).StatusCode)
}

func TestPlanWebDAVPutAndOverwrite(t *testing.T) {
	e := newPlanEnv(t, "plan-dav@example.com")
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-5)

	require.Equal(t, http.StatusCreated, davPut(e, "/webdav/a.txt", "abc"))
	assert.Equal(t, http.StatusInsufficientStorage, davPut(e, "/webdav/b.txt", "abcdef"))
	assert.Equal(t, http.StatusInsufficientStorage, davPut(e, "/webdav/a.txt", "abcdefgh"))

	resp := davRequest(e.ts, "GET", "/webdav/a.txt", e.token, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	got, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "abc", string(got))
	assert.Equal(t, http.StatusCreated, davPut(e, "/webdav/a.txt", "xy"))
	shrunk, _ := io.ReadAll(davRequest(e.ts, "GET", "/webdav/a.txt", e.token, "").Body)
	assert.Equal(t, "xy", string(shrunk))
}

func TestPlanVersionCreationIsChecked(t *testing.T) {
	e := newPlanEnv(t, "plan-version@example.com")
	e.enforce(t, 1, "1", "")
	fileID := uploadedID(t, e, "doc.txt", "abc")
	e.fill(t, nil, gibibyte-5)

	assert.Equal(t, http.StatusPaymentRequired, reuploadFile(e.ts, e.token, fileID, "abcd").StatusCode)
	assert.Equal(t, http.StatusOK, reuploadFile(e.ts, e.token, fileID, "ab").StatusCode)
}

// parallelInits starts n chunked uploads of size bytes at once and counts the
// answers by status.
func parallelInits(e planEnv, n int, size int64) map[int]int {
	var mu sync.Mutex
	var wg sync.WaitGroup
	counts := map[int]int{}
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := initChunkedUpload(e.ts, e.token, map[string]any{"file_name": fmt.Sprintf("p%d.bin", i), "total_size": size})
			mu.Lock()
			counts[resp.StatusCode]++
			mu.Unlock()
		}()
	}
	wg.Wait()
	return counts
}

func TestPlanParallelSessionsCannotStagePastThePlan(t *testing.T) {
	e := newPlanEnv(t, "plan-parallel@example.com")
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-1000)

	counts := parallelInits(e, 20, 300)
	assert.Equal(t, 3, counts[http.StatusCreated])
	assert.Equal(t, 17, counts[http.StatusPaymentRequired])
}

func TestOpenSessionsAreCappedPerUser(t *testing.T) {
	e := newPlanEnv(t, "plan-cap@example.com")

	counts := parallelInits(e, 20, 1)
	assert.Equal(t, 10, counts[http.StatusCreated])
	assert.Equal(t, 10, counts[http.StatusTooManyRequests])
}

func TestPlanIdleSessionStopsReserving(t *testing.T) {
	e := newPlanEnv(t, "plan-idle@example.com")
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-1000)

	abandoned := initSession(t, e.ts, e.token, 900)
	assert.Equal(t, http.StatusPaymentRequired,
		initChunkedUpload(e.ts, e.token, map[string]any{"file_name": "next.bin", "total_size": 200}).StatusCode)

	require.NoError(t, e.ts.db.Exec("UPDATE upload_sessions SET created_at = now() - interval '2 hours' WHERE id = ?", abandoned).Error)
	assert.Equal(t, http.StatusCreated,
		initChunkedUpload(e.ts, e.token, map[string]any{"file_name": "next.bin", "total_size": 200}).StatusCode)
}

func TestPlanLapsedSessionMustReadmitBeforeItsNextChunk(t *testing.T) {
	e := newPlanEnv(t, "plan-revive@example.com")
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-1000)
	lapsed := initSession(t, e.ts, e.token, 900)
	require.NoError(t, e.ts.db.Exec("UPDATE upload_sessions SET created_at = now() - interval '2 hours' WHERE id = ?", lapsed).Error)

	taken := initSession(t, e.ts, e.token, 200)
	assert.Equal(t, http.StatusPaymentRequired, putChunk(e.ts, e.token, lapsed, 1, make([]byte, 100)).StatusCode,
		"the room the lapsed session held went to another upload")

	require.Equal(t, http.StatusOK, doDelete(e.ts, "/files/upload/"+taken, e.token).StatusCode)
	assert.Equal(t, http.StatusCreated, putChunk(e.ts, e.token, lapsed, 1, make([]byte, 100)).StatusCode)
	assert.Equal(t, http.StatusPaymentRequired,
		initChunkedUpload(e.ts, e.token, map[string]any{"file_name": "after.bin", "total_size": 200}).StatusCode,
		"a revived session reserves again")
}

func TestLapsedSessionsStillCountTowardTheCap(t *testing.T) {
	e := newPlanEnv(t, "plan-pileup@example.com")
	for range 10 {
		initSession(t, e.ts, e.token, 1)
	}
	require.NoError(t, e.ts.db.Exec("UPDATE upload_sessions SET created_at = now() - interval '2 hours'").Error)
	assert.Equal(t, http.StatusTooManyRequests,
		initChunkedUpload(e.ts, e.token, map[string]any{"file_name": "eleventh.bin", "total_size": 1}).StatusCode)
}
