package files

import (
	"path/filepath"
	"strings"
)

const genericMime = "application/octet-stream"

var mediaTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/ogg",
	".flac": "audio/flac",
	".mp4":  "video/mp4",
	".m4v":  "video/mp4",
	".mov":  "video/quicktime",
	".webm": "video/webm",
	".ogv":  "video/ogg",
	".mkv":  "video/x-matroska",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".avif": "image/avif",
	".svg":  "image/svg+xml",
	".heic": "image/heic",
	".pdf":  "application/pdf",
}

// resolveMime returns the declared type unless it says nothing, in which case
// the extension decides. curl, sync clients and some browsers send
// application/octet-stream for media, and a stored generic type hides the file
// from every previewer. The table is explicit because the distroless runtime
// image ships no /etc/mime.types, and the standard library's guess would make a
// local build and production disagree on the same upload.
func resolveMime(name, declared string) string {
	declared = strings.TrimSpace(declared)
	if declared != "" && declared != genericMime {
		return declared
	}
	ext := strings.ToLower(filepath.Ext(name))
	if known, ok := mediaTypes[ext]; ok {
		return known
	}
	return genericMime
}
