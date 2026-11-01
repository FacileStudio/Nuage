package trash

import (
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

func TestFileObjectsIncludeThumbnailsOnlyForPossibleImages(t *testing.T) {
	cases := []struct {
		mime string
		want int
	}{
		{"image/jpeg", 2},
		{"application/octet-stream", 2},
		{"", 2},
		{"application/pdf", 1},
		{"video/mp4", 1},
	}
	for _, c := range cases {
		got := fileObjects(schemas.File{ID: 7, Bucket: "b", BucketKey: "1/abc", MimeType: c.mime})
		if len(got) != c.want {
			t.Errorf("%q: %d objects, want %d", c.mime, len(got), c.want)
		}
		if got[0].key != "1/abc" || got[0].prefix {
			t.Errorf("%q: first object must be the stored bytes, got %+v", c.mime, got[0])
		}
	}
}

func TestFileObjectsDeleteThumbnailsByPrefixInTheFilesOwnBucket(t *testing.T) {
	got := fileObjects(schemas.File{ID: 7, Bucket: "nuage-space-x", BucketKey: "1/abc", MimeType: "image/png"})

	last := got[len(got)-1]
	if !last.prefix || last.bucket != "nuage-space-x" || last.key != "thumbs/7-" {
		t.Errorf("thumbnail object = %+v, want prefix thumbs/7- in nuage-space-x", last)
	}
}
