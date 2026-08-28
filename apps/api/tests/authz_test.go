package tests

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createSpace(t *testing.T, ts *testServer, token, name string) int64 {
	t.Helper()
	resp := doJSON(ts, "POST", "/spaces", map[string]string{"name": name}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var space struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &space)
	require.NotZero(t, space.ID)
	return space.ID
}

func uploadFileToSpace(ts *testServer, token, filename, content string, spaceID int64) *http.Response {
	return uploadFileWithFields(ts, token, filename, content, map[string]string{
		"space_id": fmt.Sprintf("%d", spaceID),
	})
}

func addSpaceMember(t *testing.T, ts *testServer, token string, spaceID int64, userID string) {
	t.Helper()
	numericID, err := strconv.ParseInt(userID, 10, 64)
	require.NoError(t, err)
	resp := doJSON(ts, "POST", fmt.Sprintf("/spaces/%d/members", spaceID), map[string]any{
		"user_id": numericID,
		"role":    "member",
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func spaceMemberID(t *testing.T, ts *testServer, token string, spaceID int64, userID string) int64 {
	t.Helper()
	numericID, err := strconv.ParseInt(userID, 10, 64)
	require.NoError(t, err)

	resp := doGet(ts, fmt.Sprintf("/spaces/%d/members", spaceID), token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listing struct {
		Members []struct {
			ID     int64 `json:"id"`
			UserID int64 `json:"user_id"`
		} `json:"members"`
	}
	parseJSON(resp, &listing)

	for _, m := range listing.Members {
		if m.UserID == numericID {
			return m.ID
		}
	}
	t.Fatalf("user %s is not a member of space %d", userID, spaceID)
	return 0
}

func promoteToOwner(t *testing.T, ts *testServer, token string, spaceID int64, memberID int64) {
	t.Helper()
	resp := doJSON(ts, "PUT", fmt.Sprintf("/spaces/%d/members/%d", spaceID, memberID), map[string]any{
		"role": "owner",
	}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSoleOwnerCannotLeaveSpace(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "sole-owner-leave@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Solo Space")

	resp := doJSON(ts, "POST", fmt.Sprintf("/spaces/%d/leave", spaceID), nil, owner)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	listResp := doGet(ts, fmt.Sprintf("/spaces/%d", spaceID), owner)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
}

func TestOwnerCanLeaveOnceAnotherOwnerExists(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "first-owner-leave@example.com", "password12345")
	heirID, _ := registerUser(ts, "second-owner-leave@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Succession Space")
	addSpaceMember(t, ts, owner, spaceID, heirID)
	promoteToOwner(t, ts, owner, spaceID, spaceMemberID(t, ts, owner, spaceID, heirID))

	resp := doJSON(ts, "POST", fmt.Sprintf("/spaces/%d/leave", spaceID), nil, owner)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	afterResp := doGet(ts, fmt.Sprintf("/spaces/%d", spaceID), owner)
	assert.Equal(t, http.StatusForbidden, afterResp.StatusCode)
}

func TestMemberCanLeaveSpaceAndLosesAccess(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner-leave-access@example.com", "password12345")
	memberID, member := registerUser(ts, "member-leave-access@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Revolving Space")
	addSpaceMember(t, ts, owner, spaceID, memberID)
	require.Equal(t, http.StatusOK, doGet(ts, fmt.Sprintf("/files?space_id=%d", spaceID), member).StatusCode)

	resp := doJSON(ts, "POST", fmt.Sprintf("/spaces/%d/leave", spaceID), nil, member)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, http.StatusForbidden, doGet(ts, fmt.Sprintf("/files?space_id=%d", spaceID), member).StatusCode)
}

func TestNonMemberCannotLeaveSpace(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner-leave-outsider@example.com", "password12345")
	_, outsider := registerUser(ts, "outsider-leave@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Closed Space")

	resp := doJSON(ts, "POST", fmt.Sprintf("/spaces/%d/leave", spaceID), nil, outsider)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestCannotRemoveTheLastOwner(t *testing.T) {
	ts := setupTestServer(t)
	ownerID, owner := registerUser(ts, "owner-remove-last@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Last Owner Space")
	memberID := spaceMemberID(t, ts, owner, spaceID, ownerID)

	resp := doDelete(ts, fmt.Sprintf("/spaces/%d/members/%d", spaceID, memberID), owner)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestOwnerCanBeRemovedWhileAnotherOwnerRemains(t *testing.T) {
	ts := setupTestServer(t)
	ownerID, owner := registerUser(ts, "owner-removed@example.com", "password12345")
	heirID, heir := registerUser(ts, "owner-remover@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Two Owner Space")
	addSpaceMember(t, ts, owner, spaceID, heirID)
	promoteToOwner(t, ts, owner, spaceID, spaceMemberID(t, ts, owner, spaceID, heirID))

	resp := doDelete(ts, fmt.Sprintf("/spaces/%d/members/%d", spaceID, spaceMemberID(t, ts, owner, spaceID, ownerID)), heir)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, http.StatusForbidden, doGet(ts, fmt.Sprintf("/spaces/%d", spaceID), owner).StatusCode)
}

func TestAnAdminCannotRemoveAnOwner(t *testing.T) {
	ts := setupTestServer(t)
	ownerID, owner := registerUser(ts, "owner-kept@example.com", "password12345")
	heirID, _ := registerUser(ts, "owner-second@example.com", "password12345")
	adminID, admin := registerUser(ts, "admin-attacker@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Two Owner Space")
	addSpaceMember(t, ts, owner, spaceID, heirID)
	promoteToOwner(t, ts, owner, spaceID, spaceMemberID(t, ts, owner, spaceID, heirID))
	addSpaceMember(t, ts, owner, spaceID, adminID)
	promoteToAdmin(t, ts, owner, spaceID, spaceMemberID(t, ts, owner, spaceID, adminID))

	resp := doDelete(ts, fmt.Sprintf("/spaces/%d/members/%d", spaceID, spaceMemberID(t, ts, owner, spaceID, ownerID)), admin)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	assert.Equal(t, http.StatusOK, doGet(ts, fmt.Sprintf("/spaces/%d", spaceID), owner).StatusCode)
}

func promoteToAdmin(t *testing.T, ts *testServer, token string, spaceID int64, memberID int64) {
	t.Helper()
	resp := doJSON(ts, "PUT", fmt.Sprintf("/spaces/%d/members/%d", spaceID, memberID), map[string]any{
		"role": "admin",
	}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCannotShareAnotherUsersFile(t *testing.T) {
	ts := setupTestServer(t)
	_, victim := registerUser(ts, "victim-share@example.com", "password12345")
	_, attacker := registerUser(ts, "attacker-share@example.com", "password12345")

	resp := uploadFile(ts, victim, "secret.txt", "confidential", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	shareResp := doJSON(ts, "POST", "/shares", map[string]any{"file_id": file.ID}, attacker)
	assert.Equal(t, http.StatusNotFound, shareResp.StatusCode)
}

func TestCannotShareAnotherUsersFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, victim := registerUser(ts, "victim-folder@example.com", "password12345")
	_, attacker := registerUser(ts, "attacker-folder@example.com", "password12345")

	resp := doJSON(ts, "POST", "/folders", map[string]string{"name": "private"}, victim)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &folder)

	shareResp := doJSON(ts, "POST", "/shares", map[string]any{"folder_id": folder.ID}, attacker)
	assert.Equal(t, http.StatusNotFound, shareResp.StatusCode)
}

func TestCannotListFilesOfForeignSpace(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner-space@example.com", "password12345")
	_, outsider := registerUser(ts, "outsider-space@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Private Space")
	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, owner, "plans.txt", "secret", spaceID).StatusCode)

	resp := doGet(ts, fmt.Sprintf("/files?space_id=%d", spaceID), outsider)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	folderResp := doGet(ts, fmt.Sprintf("/folders?space_id=%d", spaceID), outsider)
	assert.Equal(t, http.StatusForbidden, folderResp.StatusCode)
}

func TestCannotUploadIntoForeignSpace(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner-upload@example.com", "password12345")
	_, attacker := registerUser(ts, "attacker-upload@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Team Space")

	resp := uploadFileToSpace(ts, attacker, "malware.txt", "payload", spaceID)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestSpaceMemberSeesSpaceFiles(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner-member@example.com", "password12345")
	memberID, member := registerUser(ts, "member-member@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Shared Space")
	addSpaceMember(t, ts, owner, spaceID, memberID)

	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, owner, "roadmap.txt", "plans", spaceID).StatusCode)

	resp := doGet(ts, fmt.Sprintf("/files?space_id=%d", spaceID), member)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listing struct {
		Files []struct {
			Name string `json:"name"`
		} `json:"files"`
	}
	parseJSON(resp, &listing)
	require.Len(t, listing.Files, 1)
	assert.Equal(t, "roadmap.txt", listing.Files[0].Name)
}

func TestSpaceMemberCanDownloadAnotherMembersFile(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner-dl@example.com", "password12345")
	memberID, member := registerUser(ts, "member-dl@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Download Space")
	addSpaceMember(t, ts, owner, spaceID, memberID)

	uploadResp := uploadFileToSpace(ts, owner, "shared_dl.txt", "shared content", spaceID)
	require.Equal(t, http.StatusCreated, uploadResp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(uploadResp, &file)

	dlResp := doGet(ts, fmt.Sprintf("/files/%d/download", file.ID), member)
	require.Equal(t, http.StatusOK, dlResp.StatusCode)
	defer dlResp.Body.Close()
	body, _ := io.ReadAll(dlResp.Body)
	assert.Equal(t, "shared content", string(body))
}

func TestOutsiderCannotDownloadSpaceFile(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner-dl2@example.com", "password12345")
	_, outsider := registerUser(ts, "outsider-dl@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Closed Download Space")
	uploadResp := uploadFileToSpace(ts, owner, "private_dl.txt", "secret", spaceID)
	require.Equal(t, http.StatusCreated, uploadResp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(uploadResp, &file)

	dlResp := doGet(ts, fmt.Sprintf("/files/%d/download", file.ID), outsider)
	assert.Equal(t, http.StatusNotFound, dlResp.StatusCode)
}

func TestCannotUploadIntoAnotherUsersFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, victim := registerUser(ts, "victim-inject@example.com", "password12345")
	_, attacker := registerUser(ts, "attacker-inject@example.com", "password12345")

	resp := doJSON(ts, "POST", "/folders", map[string]string{"name": "victimfolder"}, victim)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &folder)

	uploadResp := uploadFile(ts, attacker, "planted.txt", "payload", &folder.ID)
	assert.Equal(t, http.StatusNotFound, uploadResp.StatusCode)
}

func TestChunkedUploadCannotTargetAnotherUsersFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, victim := registerUser(ts, "victim-chunk@example.com", "password12345")
	_, attacker := registerUser(ts, "attacker-chunk@example.com", "password12345")

	resp := doJSON(ts, "POST", "/folders", map[string]string{"name": "chunktarget"}, victim)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &folder)

	initResp := doJSON(ts, "POST", "/files/upload/init", map[string]any{
		"file_name":  "planted.bin",
		"total_size": 8,
		"folder_id":  folder.ID,
	}, attacker)
	assert.Equal(t, http.StatusNotFound, initResp.StatusCode)
}

func TestNonAdminCannotSetQuota(t *testing.T) {
	ts := setupTestServer(t)
	registerUser(ts, "first-admin@example.com", "password12345")
	targetID, _ := registerUser(ts, "plain-user@example.com", "password12345")
	_, attacker := registerUser(ts, "quota-attacker@example.com", "password12345")

	resp := doJSON(ts, "PUT", fmt.Sprintf("/quota/users/%s", targetID), map[string]any{
		"storage_limit": -1,
	}, attacker)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	listResp := doGet(ts, "/quota/users", attacker)
	assert.Equal(t, http.StatusForbidden, listResp.StatusCode)
}

func TestAdminCanSetQuota(t *testing.T) {
	ts := setupTestServer(t)
	_, admin := registerUser(ts, "real-admin@example.com", "password12345")
	targetID, _ := registerUser(ts, "quota-target@example.com", "password12345")

	resp := doJSON(ts, "PUT", fmt.Sprintf("/quota/users/%s", targetID), map[string]any{
		"storage_limit": 1024,
	}, admin)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNonAdminCannotReadInstanceActivity(t *testing.T) {
	ts := setupTestServer(t)
	_, admin := registerUser(ts, "activity-admin@example.com", "password12345")
	_, other := registerUser(ts, "activity-other@example.com", "password12345")

	require.Equal(t, http.StatusCreated, uploadFile(ts, admin, "adminfile.txt", "data", nil).StatusCode)

	resp := doGet(ts, "/activity", other)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestActivityForForeignFileIsRejected(t *testing.T) {
	ts := setupTestServer(t)
	_, victim := registerUser(ts, "activity-victim@example.com", "password12345")
	_, attacker := registerUser(ts, "activity-attacker@example.com", "password12345")

	resp := uploadFile(ts, victim, "tracked.txt", "data", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	actResp := doGet(ts, fmt.Sprintf("/activity/files/%d", file.ID), attacker)
	assert.Equal(t, http.StatusNotFound, actResp.StatusCode)
}

func TestFilenameDeduplicationIsScopedPerUser(t *testing.T) {
	ts := setupTestServer(t)
	_, first := registerUser(ts, "dedup-one@example.com", "password12345")
	_, second := registerUser(ts, "dedup-two@example.com", "password12345")

	require.Equal(t, http.StatusCreated, uploadFile(ts, first, "report.txt", "mine", nil).StatusCode)

	resp := uploadFile(ts, second, "report.txt", "theirs", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		Name string `json:"name"`
	}
	parseJSON(resp, &file)
	assert.Equal(t, "report.txt", file.Name, "another user's filename must not leak through deduplication")
}
