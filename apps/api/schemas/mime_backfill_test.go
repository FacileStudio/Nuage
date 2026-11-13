package schemas

import (
	"testing"
	"time"
)

func TestBackfillMimeTypes(t *testing.T) {
	orm := openTestDatabase(t)
	if err := orm.Exec(`INSERT INTO users (email, password_hash) VALUES ('mime@example.com', 'hash')`).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}
	var owner User
	if err := orm.Where("email = ?", "mime@example.com").First(&owner).Error; err != nil {
		t.Fatalf("read user: %v", err)
	}

	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	rows := []struct {
		name string
		mime string
		want string
	}{
		{"song.mp3", "application/octet-stream", "audio/mpeg"},
		{"CLIP.MP4", "application/octet-stream", "video/mp4"},
		{"scan.Pdf", "", "application/pdf"},
		{"photo.jpeg", "application/octet-stream", "image/jpeg"},
		{"archive.tar.mp3", "application/octet-stream", "audio/mpeg"},
		{"photo.jpg", "image/png", "image/png"},
		{"notes.mp3", "text/plain", "text/plain"},
		{"blob.bin", "application/octet-stream", "application/octet-stream"},
		{"README", "application/octet-stream", "application/octet-stream"},
		{"dir.mp3/data", "application/octet-stream", "application/octet-stream"},
	}
	for i, row := range rows {
		if err := orm.Exec(
			`INSERT INTO files (facile_id, name, mime_type, bucket_key, bucket, uploaded_by, created_at, updated_at)
			 VALUES (?, ?, ?, ?, '', ?, ?, ?)`,
			"fil_"+row.name, row.name, row.mime, "key-"+row.name, owner.ID, stamp, stamp).Error; err != nil {
			t.Fatalf("insert row %d (%s): %v", i, row.name, err)
		}
	}

	for pass := 1; pass <= 2; pass++ {
		if err := backfillMimeTypes(orm); err != nil {
			t.Fatalf("backfill pass %d: %v", pass, err)
		}
		for _, row := range rows {
			var got File
			if err := orm.Where("name = ?", row.name).First(&got).Error; err != nil {
				t.Fatalf("read %s: %v", row.name, err)
			}
			if got.MimeType != row.want {
				t.Errorf("pass %d: %s mime_type = %q, want %q", pass, row.name, got.MimeType, row.want)
			}
			if !got.UpdatedAt.Equal(stamp) {
				t.Errorf("pass %d: %s updated_at = %v, want it untouched at %v", pass, row.name, got.UpdatedAt, stamp)
			}
		}
	}
}
