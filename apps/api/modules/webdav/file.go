package webdav

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"golang.org/x/net/webdav"
)

type VirtualDir struct {
	fs       *NuageFS
	ctx      context.Context
	folderID *int64
	dirName  string
	modTime  time.Time
	children []os.FileInfo
	pos      int
}

func (d *VirtualDir) Read([]byte) (int, error)                  { return 0, os.ErrInvalid }
func (d *VirtualDir) Write([]byte) (int, error)                 { return 0, os.ErrInvalid }
func (d *VirtualDir) Seek(int64, int) (int64, error)            { return 0, os.ErrInvalid }
func (d *VirtualDir) Close() error                              { return nil }
func (d *VirtualDir) Stat() (os.FileInfo, error) {
	return &DirInfo{name: d.dirName, modTime: d.modTime}, nil
}

func (d *VirtualDir) DeadProps() (map[xml.Name]webdav.Property, error) {
	return nil, nil
}

func (d *VirtualDir) Patch(patches []webdav.Proppatch) ([]webdav.Propstat, error) {
	pstat := webdav.Propstat{Status: http.StatusOK}
	for _, patch := range patches {
		for _, p := range patch.Props {
			pstat.Props = append(pstat.Props, webdav.Property{XMLName: p.XMLName})
		}
	}
	return []webdav.Propstat{pstat}, nil
}

func (d *VirtualDir) Readdir(count int) ([]os.FileInfo, error) {
	if d.children == nil {
		var folders []schemas.Folder
		fq := d.fs.db.WithContext(d.ctx).Where("owner_id = ? AND deleted_at IS NULL", d.fs.userID).Order("name asc")
		if d.folderID != nil {
			fq = fq.Where("parent_id = ?", *d.folderID)
		} else {
			fq = fq.Where("parent_id IS NULL")
		}
		fq.Find(&folders)

		var files []schemas.File
		ffq := d.fs.db.WithContext(d.ctx).Where("uploaded_by = ? AND deleted_at IS NULL", d.fs.userID).Order("name asc")
		if d.folderID != nil {
			ffq = ffq.Where("folder_id = ?", *d.folderID)
		} else {
			ffq = ffq.Where("folder_id IS NULL")
		}
		ffq.Find(&files)

		d.children = make([]os.FileInfo, 0, len(folders)+len(files))
		for _, f := range folders {
			d.children = append(d.children, &DirInfo{name: f.Name, modTime: f.UpdatedAt})
		}
		for _, f := range files {
			d.children = append(d.children, &nuageFileInfo{
				name: f.Name, size: f.Size, modTime: f.UpdatedAt, mimeType: f.MimeType,
			})
		}
	}

	if count <= 0 {
		result := d.children[d.pos:]
		d.pos = len(d.children)
		return result, nil
	}

	if d.pos >= len(d.children) {
		return nil, io.EOF
	}
	end := d.pos + count
	if end > len(d.children) {
		end = len(d.children)
	}
	result := d.children[d.pos:end]
	d.pos = end
	if d.pos >= len(d.children) {
		return result, io.EOF
	}
	return result, nil
}

type VirtualFile struct {
	fs         *NuageFS
	ctx        context.Context
	file       *schemas.File
	name       string
	parentPath string
	reader     *bytes.Reader
	buf        *bytes.Buffer
	writable   bool
	creating   bool
	closed     bool
}

func (f *VirtualFile) Read(p []byte) (int, error) {
	if f.reader == nil {
		return 0, io.EOF
	}
	return f.reader.Read(p)
}

func (f *VirtualFile) Write(p []byte) (int, error) {
	if !f.writable {
		return 0, os.ErrPermission
	}
	return f.buf.Write(p)
}

func (f *VirtualFile) Seek(offset int64, whence int) (int64, error) {
	if f.reader != nil {
		return f.reader.Seek(offset, whence)
	}
	return 0, nil
}

func (f *VirtualFile) Readdir(int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (f *VirtualFile) Stat() (os.FileInfo, error) {
	if f.file != nil {
		return &nuageFileInfo{
			name: f.file.Name, size: f.file.Size,
			modTime: f.file.UpdatedAt, mimeType: f.file.MimeType,
		}, nil
	}
	size := int64(0)
	if f.buf != nil {
		size = int64(f.buf.Len())
	}
	return &nuageFileInfo{name: f.name, size: size, modTime: time.Now()}, nil
}

func (f *VirtualFile) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true

	if !f.writable || f.buf == nil || f.buf.Len() == 0 {
		return nil
	}

	data := f.buf.Bytes()

	if f.creating {
		return f.fs.createFile(f.ctx, f.name, f.parentPath, data)
	}
	if f.file != nil {
		return f.fs.overwriteFile(f.ctx, f.file, data)
	}
	return nil
}

func (f *VirtualFile) ContentType(_ context.Context) (string, error) {
	if f.file != nil && f.file.MimeType != "" {
		return f.file.MimeType, nil
	}
	return "", ErrNotImplemented
}

func (f *VirtualFile) DeadProps() (map[xml.Name]webdav.Property, error) {
	return nil, nil
}

func (f *VirtualFile) Patch(patches []webdav.Proppatch) ([]webdav.Propstat, error) {
	pstat := webdav.Propstat{Status: http.StatusOK}
	for _, patch := range patches {
		for _, p := range patch.Props {
			pstat.Props = append(pstat.Props, webdav.Property{XMLName: p.XMLName})
		}
	}
	return []webdav.Propstat{pstat}, nil
}

type nuageFileInfo struct {
	name     string
	size     int64
	modTime  time.Time
	mimeType string
}

func (fi *nuageFileInfo) Name() string      { return fi.name }
func (fi *nuageFileInfo) Size() int64       { return fi.size }
func (fi *nuageFileInfo) Mode() os.FileMode { return 0644 }
func (fi *nuageFileInfo) ModTime() time.Time { return fi.modTime.UTC() }
func (fi *nuageFileInfo) IsDir() bool       { return false }
func (fi *nuageFileInfo) Sys() any          { return nil }

func (fi *nuageFileInfo) ContentType(_ context.Context) (string, error) {
	if fi.mimeType != "" {
		return fi.mimeType, nil
	}
	return "", ErrNotImplemented
}

type DirInfo struct {
	name    string
	modTime time.Time
}

func (di *DirInfo) Name() string      { return di.name }
func (di *DirInfo) Size() int64       { return 0 }
func (di *DirInfo) Mode() os.FileMode { return os.ModeDir | 0755 }
func (di *DirInfo) ModTime() time.Time { return di.modTime.UTC() }
func (di *DirInfo) IsDir() bool       { return true }
func (di *DirInfo) Sys() any          { return nil }

type DevNullFile struct {
	name string
}

func (d *DevNullFile) Read([]byte) (int, error)       { return 0, io.EOF }
func (d *DevNullFile) Write(p []byte) (int, error)    { return len(p), nil }
func (d *DevNullFile) Seek(int64, int) (int64, error) { return 0, nil }
func (d *DevNullFile) Readdir(int) ([]os.FileInfo, error) { return nil, os.ErrInvalid }
func (d *DevNullFile) Close() error                   { return nil }
func (d *DevNullFile) Stat() (os.FileInfo, error) {
	return &nuageFileInfo{name: d.name, size: 0, modTime: time.Now()}, nil
}

func (d *DevNullFile) DeadProps() (map[xml.Name]webdav.Property, error) {
	return nil, nil
}

func (d *DevNullFile) Patch(patches []webdav.Proppatch) ([]webdav.Propstat, error) {
	pstat := webdav.Propstat{Status: http.StatusOK}
	for _, patch := range patches {
		for _, p := range patch.Props {
			pstat.Props = append(pstat.Props, webdav.Property{XMLName: p.XMLName})
		}
	}
	return []webdav.Propstat{pstat}, nil
}

var ErrNotImplemented = errors.New("not implemented")
