package webdav

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"golang.org/x/net/webdav"
	"gorm.io/gorm"
)

type NuageFS struct {
	db      *gorm.DB
	storage *storage.Client
	quota   *quota.Service
	userID  int64
}

func NewNuageFS(db *gorm.DB, storageClient *storage.Client, quotaService *quota.Service, userID int64) webdav.FileSystem {
	return &NuageFS{db: db, storage: storageClient, quota: quotaService, userID: userID}
}

func (fs *NuageFS) resolvePath(ctx context.Context, name string) (*schemas.Folder, *schemas.File, error) {
	name = path.Clean(name)
	if name == "/" || name == "." || name == "" {
		return nil, nil, nil
	}

	parts := strings.Split(strings.Trim(name, "/"), "/")
	var currentFolderID *int64

	for i, part := range parts {
		isLast := i == len(parts)-1

		var folder schemas.Folder
		q := fs.db.WithContext(ctx).Where("name = ? AND owner_id = ? AND deleted_at IS NULL", part, fs.userID)
		if currentFolderID != nil {
			q = q.Where("parent_id = ?", *currentFolderID)
		} else {
			q = q.Where("parent_id IS NULL")
		}

		if err := q.First(&folder).Error; err == nil {
			if isLast {
				return &folder, nil, nil
			}
			currentFolderID = &folder.ID
			continue
		}

		if isLast {
			var file schemas.File
			fq := fs.db.WithContext(ctx).Where("name = ? AND uploaded_by = ? AND deleted_at IS NULL", part, fs.userID)
			if currentFolderID != nil {
				fq = fq.Where("folder_id = ?", *currentFolderID)
			} else {
				fq = fq.Where("folder_id IS NULL")
			}
			if err := fq.First(&file).Error; err == nil {
				return nil, &file, nil
			}
		}

		return nil, nil, os.ErrNotExist
	}

	return nil, nil, os.ErrNotExist
}

func (fs *NuageFS) resolveParentFolder(ctx context.Context, name string) (*int64, error) {
	dir := path.Dir(name)
	if dir == "/" || dir == "." || dir == "" {
		return nil, nil
	}
	parent, _, err := fs.resolvePath(ctx, dir)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, os.ErrNotExist
	}
	return &parent.ID, nil
}

func (fs *NuageFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	name = path.Clean(name)
	if name == "/" || name == "." || name == "" {
		return &DirInfo{name: "/", modTime: time.Now()}, nil
	}

	folder, file, err := fs.resolvePath(ctx, name)
	if err != nil {
		if isJunkFile(name) {
			return &nuageFileInfo{name: path.Base(name), size: 0, modTime: time.Now()}, nil
		}
		return nil, err
	}
	if folder != nil {
		return &DirInfo{name: folder.Name, modTime: folder.UpdatedAt}, nil
	}
	if file != nil {
		return &nuageFileInfo{name: file.Name, size: file.Size, modTime: file.UpdatedAt, mimeType: file.MimeType}, nil
	}
	return nil, os.ErrNotExist
}

func (fs *NuageFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	name = path.Clean(name)
	base := path.Base(name)

	if _, _, err := fs.resolvePath(ctx, name); err == nil {
		return os.ErrExist
	}

	parentID, err := fs.resolveParentFolder(ctx, name)
	if err != nil {
		return err
	}

	folder := &schemas.Folder{
		FacileID: facile.NewID(),
		Name:     base,
		ParentID: parentID,
		OwnerID:  fs.userID,
	}
	return fs.db.WithContext(ctx).Create(folder).Error
}

func (fs *NuageFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	name = path.Clean(name)

	if name == "/" || name == "." || name == "" {
		return &VirtualDir{fs: fs, ctx: ctx, folderID: nil, dirName: "/"}, nil
	}

	folder, file, err := fs.resolvePath(ctx, name)

	if folder != nil {
		return &VirtualDir{fs: fs, ctx: ctx, folderID: &folder.ID, dirName: folder.Name, modTime: folder.UpdatedAt}, nil
	}

	if file != nil {
		if flag&os.O_TRUNC != 0 {
			return &VirtualFile{
				fs: fs, ctx: ctx, file: file, name: file.Name,
				writable: true,
			}, nil
		}
		rc, sErr := fs.storage.GetObject(ctx, file.BucketKey)
		if sErr != nil {
			return nil, sErr
		}
		if rs, ok := rc.(io.ReadSeeker); ok {
			return &VirtualFile{
				fs: fs, ctx: ctx, file: file, name: file.Name,
				reader: rs, closer: rc,
			}, nil
		}
		tmp, sErr := spoolToTemp(rc)
		rc.Close()
		if sErr != nil {
			return nil, sErr
		}
		return &VirtualFile{
			fs: fs, ctx: ctx, file: file, name: file.Name,
			reader: tmp, closer: &removeOnClose{tmp},
		}, nil
	}

	if isJunkFile(name) {
		return &DevNullFile{name: path.Base(name)}, nil
	}

	if flag&os.O_CREATE != 0 {
		if err != nil && err != os.ErrNotExist {
			return nil, err
		}
		return &VirtualFile{
			fs: fs, ctx: ctx, name: path.Base(name), parentPath: path.Dir(name),
			writable: true, creating: true,
		}, nil
	}

	return nil, os.ErrNotExist
}

func (fs *NuageFS) RemoveAll(ctx context.Context, name string) error {
	name = path.Clean(name)
	if name == "/" || name == "." {
		return os.ErrPermission
	}

	folder, file, err := fs.resolvePath(ctx, name)
	if err != nil {
		return err
	}

	now := time.Now()

	if file != nil {
		if file.UploadedBy != fs.userID {
			return os.ErrPermission
		}
		return fs.db.WithContext(ctx).Model(file).Update("deleted_at", now).Error
	}
	if folder != nil {
		if folder.OwnerID != fs.userID {
			return os.ErrPermission
		}
		return fs.softDeleteRecursive(ctx, folder.ID, now)
	}
	return os.ErrNotExist
}

func (fs *NuageFS) softDeleteRecursive(ctx context.Context, folderID int64, now time.Time) error {
	var subfolders []schemas.Folder
	fs.db.WithContext(ctx).Where("parent_id = ? AND owner_id = ? AND deleted_at IS NULL", folderID, fs.userID).Find(&subfolders)
	for _, sub := range subfolders {
		if err := fs.softDeleteRecursive(ctx, sub.ID, now); err != nil {
			return err
		}
	}

	if err := fs.db.WithContext(ctx).Model(&schemas.File{}).
		Where("folder_id = ? AND uploaded_by = ? AND deleted_at IS NULL", folderID, fs.userID).
		Update("deleted_at", now).Error; err != nil {
		return err
	}

	return fs.db.WithContext(ctx).Model(&schemas.Folder{}).
		Where("id = ?", folderID).
		Update("deleted_at", now).Error
}

func (fs *NuageFS) Rename(ctx context.Context, oldName, newName string) error {
	oldName = path.Clean(oldName)
	newName = path.Clean(newName)

	folder, file, err := fs.resolvePath(ctx, oldName)
	if err != nil {
		return err
	}

	if file != nil && file.UploadedBy != fs.userID {
		return os.ErrPermission
	}
	if folder != nil && folder.OwnerID != fs.userID {
		return os.ErrPermission
	}

	newBase := path.Base(newName)
	var newParentID *int64
	newDir := path.Dir(newName)
	if newDir != "/" && newDir != "." {
		parent, _, pErr := fs.resolvePath(ctx, newDir)
		if pErr != nil {
			return pErr
		}
		if parent == nil {
			return os.ErrNotExist
		}
		if parent.OwnerID != fs.userID {
			return os.ErrPermission
		}
		newParentID = &parent.ID
	}

	if file != nil {
		updates := map[string]any{"name": newBase}
		if newParentID != nil {
			updates["folder_id"] = *newParentID
		} else {
			updates["folder_id"] = nil
		}
		return fs.db.WithContext(ctx).Model(file).Updates(updates).Error
	}
	if folder != nil {
		updates := map[string]any{"name": newBase}
		if newParentID != nil {
			updates["parent_id"] = *newParentID
		} else {
			updates["parent_id"] = nil
		}
		return fs.db.WithContext(ctx).Model(folder).Updates(updates).Error
	}
	return os.ErrNotExist
}

func (fs *NuageFS) createFile(ctx context.Context, name string, parentPath string, content io.ReadSeeker, size int64) error {
	var folderID *int64
	if parentPath != "/" && parentPath != "." && parentPath != "" {
		parent, _, err := fs.resolvePath(ctx, parentPath)
		if err != nil {
			return err
		}
		if parent != nil {
			folderID = &parent.ID
		}
	}

	if fs.quota != nil {
		if err := fs.quota.CheckQuota(ctx, fs.userID, size); err != nil {
			return err
		}
	}

	mimeType, err := detectMimeType(content)
	if err != nil {
		return err
	}

	fid := facile.NewID()
	bucketKey := fmt.Sprintf("%d/%s/%s", fs.userID, fid, name)

	hasher := sha256.New()
	tee := io.TeeReader(content, hasher)
	if err := fs.storage.PutObject(ctx, bucketKey, tee, size, mimeType); err != nil {
		return err
	}

	record := &schemas.File{
		FacileID:   fid,
		Name:       name,
		MimeType:   mimeType,
		Size:       size,
		Hash:       hex.EncodeToString(hasher.Sum(nil)),
		BucketKey:  bucketKey,
		FolderID:   folderID,
		UploadedBy: fs.userID,
	}
	if err := fs.db.WithContext(ctx).Create(record).Error; err != nil {
		_ = fs.storage.DeleteObject(ctx, bucketKey)
		return err
	}
	if fs.quota != nil {
		fs.quota.UpdateUsage(ctx, fs.userID, size)
	}
	return nil
}

func (fs *NuageFS) overwriteFile(ctx context.Context, file *schemas.File, content io.ReadSeeker, size int64) error {
	if file.UploadedBy != fs.userID {
		return os.ErrPermission
	}

	if fs.quota != nil {
		if delta := size - file.Size; delta > 0 {
			if err := fs.quota.CheckQuota(ctx, fs.userID, delta); err != nil {
				return err
			}
		}
	}

	mimeType, err := detectMimeType(content)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	tee := io.TeeReader(content, hasher)
	if err := fs.storage.PutObject(ctx, file.BucketKey, tee, size, mimeType); err != nil {
		return err
	}

	if err := fs.db.WithContext(ctx).Model(file).Updates(map[string]any{
		"size":      size,
		"hash":      hex.EncodeToString(hasher.Sum(nil)),
		"mime_type": mimeType,
	}).Error; err != nil {
		return err
	}
	if fs.quota != nil {
		fs.quota.UpdateUsage(ctx, fs.userID, size-file.Size)
	}
	return nil
}

func detectMimeType(content io.ReadSeeker) (string, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(content, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", err
	}
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(head[:n]), nil
}

func isJunkFile(name string) bool {
	base := path.Base(name)
	if base == ".DS_Store" || base == ".localized" || base == "Thumbs.db" || base == "desktop.ini" {
		return true
	}
	if strings.HasPrefix(base, "._") {
		return true
	}
	return false
}
