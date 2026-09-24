package webdav

import (
	"encoding/xml"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/webdav"
)

// spacesDir is the read-only directory at the root of the spaces index. Its
// children are computed once and it is never writable.
type spacesDir struct {
	children []os.FileInfo
	modTime  time.Time
	pos      int
}

func (d *spacesDir) Read([]byte) (int, error)       { return 0, os.ErrInvalid }
func (d *spacesDir) Write([]byte) (int, error)      { return 0, os.ErrPermission }
func (d *spacesDir) Seek(int64, int) (int64, error) { return 0, os.ErrInvalid }
func (d *spacesDir) Close() error                   { return nil }

func (d *spacesDir) Stat() (os.FileInfo, error) {
	return &DirInfo{name: "/", modTime: d.modTime}, nil
}

func (d *spacesDir) Readdir(count int) ([]os.FileInfo, error) {
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

func (d *spacesDir) DeadProps() (map[xml.Name]webdav.Property, error) {
	return nil, nil
}

func (d *spacesDir) Patch(patches []webdav.Proppatch) ([]webdav.Propstat, error) {
	pstat := webdav.Propstat{Status: http.StatusOK}
	for _, patch := range patches {
		for _, p := range patch.Props {
			pstat.Props = append(pstat.Props, webdav.Property{XMLName: p.XMLName})
		}
	}
	return []webdav.Propstat{pstat}, nil
}
