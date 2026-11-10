package mcp

import (
	"bytes"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// textTypes are the media types read as text besides text/*.
var textTypes = []string{"json", "xml", "yaml", "toml", "javascript", "ecmascript", "csv", "markdown", "sql", "x-sh", "x-httpd-php", "graphql", "ld+json", "svg"}

// textExtensions name text files whatever type their uploader declared.
var textExtensions = map[string]string{
	".txt": "text/plain", ".md": "text/markdown", ".markdown": "text/markdown", ".csv": "text/csv",
	".tsv": "text/tab-separated-values", ".json": "application/json", ".yaml": "application/yaml",
	".yml": "application/yaml", ".toml": "application/toml", ".xml": "application/xml", ".html": "text/html",
	".htm": "text/html", ".css": "text/css", ".js": "text/javascript", ".ts": "text/plain", ".go": "text/plain",
	".py": "text/plain", ".rs": "text/plain", ".sh": "text/plain", ".sql": "text/plain", ".log": "text/plain",
	".ini": "text/plain", ".env": "text/plain", ".svelte": "text/plain", ".tex": "text/plain", ".ics": "text/calendar",
	".vcf": "text/vcard", ".srt": "text/plain", ".vtt": "text/vtt", ".svg": "image/svg+xml",
}

// declaresText reports whether a media type or a file name says text.
func declaresText(mimeType, name string) bool {
	mimeType = strings.ToLower(mimeType)
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	for _, kind := range textTypes {
		if strings.Contains(mimeType, kind) {
			return true
		}
	}
	_, ok := textExtensions[strings.ToLower(filepath.Ext(name))]
	return ok
}

// looksText reports whether a sample of a file reads as UTF-8 text: no NUL
// byte, and valid UTF-8 except for a rune cut at the end of the sample.
func looksText(sample []byte) bool {
	if bytes.IndexByte(sample, 0) >= 0 {
		return false
	}
	return utf8.Valid(runeAligned(sample))
}

// runeAligned drops a rune cut at the end of b.
func runeAligned(b []byte) []byte {
	for cut := 1; cut <= utf8.UTFMax && cut <= len(b); cut++ {
		r := b[len(b)-cut]
		if utf8.RuneStart(r) {
			if !utf8.FullRune(b[len(b)-cut:]) {
				return b[:len(b)-cut]
			}
			return b
		}
	}
	return b
}

// leadingContinuation counts the bytes at the start of b that continue a
// rune begun before it.
func leadingContinuation(b []byte) int {
	n := 0
	for n < len(b) && n < utf8.UTFMax && !utf8.RuneStart(b[n]) {
		n++
	}
	return n
}

// savedType is the media type a saved file gets: the extension's text type
// for text content, else Nuage's usual resolution.
func savedType(name string, text bool, resolved string) string {
	if known, ok := textExtensions[strings.ToLower(filepath.Ext(name))]; ok && text {
		return known + "; charset=utf-8"
	}
	if text && resolved == "application/octet-stream" {
		return "text/plain; charset=utf-8"
	}
	return resolved
}
