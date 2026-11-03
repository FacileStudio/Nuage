package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/spaces"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/require"
)

func spaceBucketName(t *testing.T, ts *testServer, spaceID int64) string {
	t.Helper()
	var space schemas.Space
	require.NoError(t, ts.db.First(&space, spaceID).Error)
	return storage.SpaceBucket(space.FacileID)
}

func TestSpaceFilesLiveInTheSpaceBucket(t *testing.T) {
	ts := setupTestServer(t)
	store := newTestStorageClient(t)
	ctx := context.Background()
	_, token := registerUser(ts, "bucket-owner@example.com", "password12345")

	spaceID := createSpace(t, ts, token, "Contracts")
	bucket := spaceBucketName(t, ts, spaceID)
	_, err := store.For(bucket).ListObjects(ctx, "")
	require.NoError(t, err, "creating a space creates its bucket")

	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, token, "deal.txt", "signed", spaceID).StatusCode)
	require.Equal(t, http.StatusCreated, uploadFile(ts, token, "mine.txt", "private", nil).StatusCode)

	var spaceFile, personalFile schemas.File
	require.NoError(t, ts.db.Where("name = ?", "deal.txt").First(&spaceFile).Error)
	require.NoError(t, ts.db.Where("name = ?", "mine.txt").First(&personalFile).Error)
	require.Equal(t, bucket, spaceFile.Bucket)
	require.Empty(t, personalFile.Bucket)

	_, err = store.For(bucket).StatObject(ctx, spaceFile.BucketKey)
	require.NoError(t, err, "the space file is in the space bucket")
	_, err = store.StatObject(ctx, spaceFile.BucketKey)
	require.Error(t, err, "and not in the shared bucket")

	download := doGet(ts, fmt.Sprintf("/files/%d/download", spaceFile.ID), token)
	require.Equal(t, http.StatusOK, download.StatusCode)
}

func TestDeletingASpaceRemovesItsBucket(t *testing.T) {
	ts := setupTestServer(t)
	store := newTestStorageClient(t)
	_, token := registerUser(ts, "bucket-delete@example.com", "password12345")

	spaceID := createSpace(t, ts, token, "Temporary")
	bucket := spaceBucketName(t, ts, spaceID)

	resp := doDelete(ts, fmt.Sprintf("/spaces/%d", spaceID), token)
	require.Less(t, resp.StatusCode, 300)

	_, err := store.For(bucket).ListObjects(context.Background(), "")
	require.Error(t, err, "the bucket is gone with the space")
}

func TestMigrateBucketsMovesLegacySpaceFilesOnce(t *testing.T) {
	ts := setupTestServer(t)
	store := newTestStorageClient(t)
	ctx := context.Background()
	userID, token := registerUser(ts, "bucket-migrate@example.com", "password12345")

	spaceID := createSpace(t, ts, token, "Legacy")
	bucket := spaceBucketName(t, ts, spaceID)
	var owner int64
	_, err := fmt.Sscan(userID, &owner)
	require.NoError(t, err)

	key := fmt.Sprintf("%d/fac_legacy", owner)
	reader := strings.NewReader("old bytes")
	require.NoError(t, store.PutObject(ctx, key, reader, int64(len("old bytes")), "text/plain"))
	legacy := schemas.File{FacileID: "fac_legacy", Name: "legacy.txt", BucketKey: key, UploadedBy: owner, SpaceID: &spaceID}
	require.NoError(t, ts.db.Create(&legacy).Error)

	service := spaces.NewService(ts.db, store, "")
	for range 2 {
		require.NoError(t, service.MigrateBuckets(ctx))
	}

	require.NoError(t, ts.db.First(&legacy, legacy.ID).Error)
	require.Equal(t, bucket, legacy.Bucket)
	_, err = store.For(bucket).StatObject(ctx, key)
	require.NoError(t, err, "the object moved into the space bucket")
	_, err = store.StatObject(ctx, key)
	require.Error(t, err, "and left the shared bucket")
}
