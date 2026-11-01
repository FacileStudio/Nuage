// Package mimetype maps file extensions to the media types Nuage can preview.
// It is explicit rather than reading /etc/mime.types because the distroless
// runtime image ships none, and the standard library's guess would make a
// local build and production disagree on the same upload.
package mimetype

import (
	"path/filepath"
	"strings"
)

// Generic is the type a client sends when it knows nothing about the file.
const Generic = "application/octet-stream"

var byExtension = map[string]string{
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

// Resolve returns the declared type unless it says nothing, in which case the
// extension decides. curl, sync clients and some browsers send
// application/octet-stream for media, and a stored generic type hides the file
// from every previewer.
func Resolve(name, declared string) string {
	declared = strings.TrimSpace(declared)
	if declared != "" && declared != Generic {
		return declared
	}
	if known, ok := byExtension[strings.ToLower(filepath.Ext(name))]; ok {
		return known
	}
	return Generic
}

// Extensions returns a copy of the extension table, keyed by lowercase
// extension with its leading dot.
func Extensions() map[string]string {
	table := make(map[string]string, len(byExtension))
	for ext, mediaType := range byExtension {
		table[ext] = mediaType
	}
	return table
}
