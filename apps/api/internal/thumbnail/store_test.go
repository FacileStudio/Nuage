package thumbnail

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

func TestWidthAcceptsOnlyTheTwoSizes(t *testing.T) {
	cases := map[string]struct {
		want int
		ok   bool
	}{
		"/f":         {320, true},
		"/f?w=320":   {320, true},
		"/f?w=640":   {640, true},
		"/f?w=100":   {0, false},
		"/f?w=huge":  {0, false},
		"/f?w=-640":  {0, false},
		"/f?w=64000": {0, false},
	}
	for target, c := range cases {
		got, err := Width(httptest.NewRequest("GET", target, nil))
		if (err == nil) != c.ok || (c.ok && got != c.want) {
			t.Errorf("Width(%q) = %d, %v; want %d, ok=%v", target, got, err, c.want, c.ok)
		}
	}
}

func TestThumbnailableFollowsTheResolvedImageType(t *testing.T) {
	cases := []struct {
		name string
		mime string
		size int64
		want bool
	}{
		{"photo.jpg", "image/jpeg", 1 << 20, true},
		{"photo.JPG", "application/octet-stream", 1 << 20, true},
		{"scan.pdf", "application/pdf", 1 << 20, false},
		{"logo.svg", "image/svg+xml", 1 << 10, false},
		{"huge.png", "image/png", MaxBytes + 1, false},
		{"notes.txt", "text/plain", 10, false},
	}
	for _, c := range cases {
		record := &schemas.File{Name: c.name, MimeType: c.mime, Size: c.size}
		if got := Thumbnailable(record); got != c.want {
			t.Errorf("Thumbnailable(%s, %s, %d) = %v, want %v", c.name, c.mime, c.size, got, c.want)
		}
	}
}

func TestObjectKeyChangesWithContentAndStaysUnderTheFilePrefix(t *testing.T) {
	record := &schemas.File{ID: 12, Hash: "abcdef0123456789abcdef"}
	key, etag := ObjectKey(record, 320)

	if !strings.HasPrefix(key, Prefix(12)) {
		t.Errorf("key %q is not under the prefix a purge deletes", key)
	}
	if strings.HasPrefix(key, Prefix(1)) {
		t.Errorf("key %q would be deleted when file 1 is purged", key)
	}
	other, _ := ObjectKey(&schemas.File{ID: 12, Hash: "ffffffffffffffffffff"}, 320)
	if other == key {
		t.Errorf("new content kept the old thumbnail key %q", key)
	}
	if etag != `"abcdef0123456789-320"` {
		t.Errorf("etag = %s", etag)
	}
}
