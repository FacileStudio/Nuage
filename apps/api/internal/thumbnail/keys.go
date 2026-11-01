package thumbnail

import "fmt"

// Prefix is the object-key prefix that every stored thumbnail of a file shares,
// whatever its width or content version. The trailing dash keeps file 12 from
// matching the thumbnails of file 123.
func Prefix(fileID int64) string {
	return fmt.Sprintf("thumbs/%d-", fileID)
}

// Key is the object key of one stored thumbnail. The version is the file's
// content hash, so replacing a file's content yields a new key instead of a
// stale hit.
func Key(fileID int64, version string, width int) string {
	return fmt.Sprintf("%s%s-%d.jpg", Prefix(fileID), version, width)
}
