package webdav

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/mimetype"
	"github.com/FacileStudio/Nuage/apps/api/internal/spacebucket"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

// createFile stores a new file from a PUT or the target of a COPY. The scope's
// plan is checked before the upload and again in the transaction that records
// the file, so two writes racing at the ceiling cannot both land.
func (fs *NuageFS) createFile(ctx context.Context, name string, parentPath string, content io.ReadSeeker, size int64) error {
	folderID, err := fs.parentID(ctx, parentPath)
	if err != nil {
		return err
	}
	if err := fs.quota.Admit(ctx, fs.scope.userID, fs.scope.spaceID, size); err != nil {
		return fs.refuse(err)
	}
	store, bucket, err := spacebucket.Open(ctx, fs.db, fs.storage, fs.scope.spaceID)
	if err != nil {
		return err
	}
	fid := facile.NewID()
	bucketKey := fmt.Sprintf("%d/%s/%s", fs.scope.userID, fid, name)
	obj, err := putObject(ctx, store, object{key: bucketKey, name: name, size: size}, content)
	if err != nil {
		return err
	}
	record := &schemas.File{
		FacileID: fid, Name: name, MimeType: obj.mimeType, Size: size, Hash: obj.hash,
		BucketKey: bucketKey, Bucket: bucket, FolderID: folderID,
		UploadedBy: fs.scope.userID, SpaceID: fs.scope.spaceID,
	}
	if err := fs.record(ctx, size, func(tx *gorm.DB) error { return tx.Create(record).Error }); err != nil {
		_ = store.DeleteObject(ctx, bucketKey)
		return fs.refuse(err)
	}
	fs.quota.UpdateUsage(ctx, fs.scope.userID, size)
	return nil
}

// overwriteFile replaces a file's content. The new bytes go to a new object
// and the row switches to it only once the plan admits the growth, so a
// refusal leaves the old content intact.
func (fs *NuageFS) overwriteFile(ctx context.Context, file *schemas.File, content io.ReadSeeker, size int64) error {
	if !fs.scope.ownsFile(file) {
		return os.ErrPermission
	}
	oldKey, oldSize, growth := file.BucketKey, file.Size, max(size-file.Size, 0)
	if err := fs.quota.Admit(ctx, fs.scope.userID, fs.scope.spaceID, growth); err != nil {
		return fs.refuse(err)
	}
	store := fs.storage.For(file.Bucket)
	bucketKey := fmt.Sprintf("%d/%s/%s", fs.scope.userID, facile.NewID(), file.Name)
	obj, err := putObject(ctx, store, object{key: bucketKey, name: file.Name, size: size}, content)
	if err != nil {
		return err
	}
	updates := map[string]any{"size": size, "hash": obj.hash, "mime_type": obj.mimeType, "bucket_key": bucketKey}
	if err := fs.record(ctx, growth, func(tx *gorm.DB) error { return tx.Model(file).Updates(updates).Error }); err != nil {
		_ = store.DeleteObject(ctx, bucketKey)
		return fs.refuse(err)
	}
	fs.dropReplaced(ctx, file.Bucket, oldKey)
	fs.quota.UpdateUsage(ctx, fs.scope.userID, size-oldSize)
	return nil
}

// record runs write in a transaction that first re-checks the plan for adding
// bytes under the scope's lock.
func (fs *NuageFS) record(ctx context.Context, adding int64, write func(tx *gorm.DB) error) error {
	return fs.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := fs.quota.AdmitTx(tx, fs.scope.userID, fs.scope.spaceID, adding); err != nil {
			return err
		}
		return write(tx)
	})
}

// dropReplaced deletes the object an overwrite replaced, unless a saved
// version still points at it.
func (fs *NuageFS) dropReplaced(ctx context.Context, bucket, bucketKey string) {
	var kept int64
	if err := fs.db.WithContext(ctx).Model(&schemas.FileVersion{}).Where("bucket_key = ?", bucketKey).Count(&kept).Error; err != nil || kept > 0 {
		return
	}
	_ = fs.storage.For(bucket).DeleteObject(ctx, bucketKey)
}

// object is a stored file's bytes: where they go, under which name, and what
// putObject learns while storing them.
type object struct {
	key      string
	name     string
	size     int64
	mimeType string
	hash     string
}

// putObject sniffs the media type, uploads content as the object and returns
// it with its type and SHA-256 filled in.
func putObject(ctx context.Context, store *storage.Client, obj object, content io.ReadSeeker) (object, error) {
	sniffed, err := detectMimeType(content)
	if err != nil {
		return obj, err
	}
	obj.mimeType = mimetype.Resolve(obj.name, sniffed)
	hasher := sha256.New()
	if err := store.PutObject(ctx, obj.key, io.TeeReader(content, hasher), obj.size, obj.mimeType); err != nil {
		return obj, err
	}
	obj.hash = hex.EncodeToString(hasher.Sum(nil))
	return obj, nil
}

// parentID resolves the folder a new file lands in, nil at the mount's root.
func (fs *NuageFS) parentID(ctx context.Context, parentPath string) (*int64, error) {
	parent, _, err := fs.resolvePath(ctx, parentPath)
	if err != nil || parent == nil {
		return nil, err
	}
	return &parent.ID, nil
}
