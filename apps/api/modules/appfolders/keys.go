package appfolders

import (
	"fmt"
	"io"
	"mime"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/FacileStudio/tronc/errors"
)

const (
	// MaxObjectSize is the largest object one PUT may store, the same cap as
	// a single-request upload to /files.
	MaxObjectSize    int64 = 100 << 20
	maxKeyBytes            = 512
	maxNameBytes           = 255
	maxListLimit           = 1000
	defaultListLimit       = 100
	genericMime            = "application/octet-stream"
)

// validKey refuses a key that is empty, too long, not UTF-8, holds a control
// character or a backslash, starts or ends with "/", or has an empty, "." or
// ".." segment. Keys are opaque to the server, but they are shown to users
// and may become paths in a later export, so they must read as one.
func validKey(key string) error {
	if key == "" || len(key) > maxKeyBytes || !utf8.ValidString(key) {
		return errors.Invalid("key must be 1 to 512 bytes of UTF-8")
	}
	if strings.IndexFunc(key, unicode.IsControl) >= 0 || strings.ContainsRune(key, '\\') {
		return errors.Invalid("key must not hold control characters or backslashes")
	}
	for segment := range strings.SplitSeq(key, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return errors.Invalid("key segments must not be empty, . or ..")
		}
	}
	return nil
}

// validPrefix accepts an empty prefix or one shaped like the start of a key.
func validPrefix(prefix string) error {
	if prefix == "" {
		return nil
	}
	if len(prefix) > maxKeyBytes || !utf8.ValidString(prefix) || strings.IndexFunc(prefix, unicode.IsControl) >= 0 {
		return errors.Invalid("prefix must be at most 512 bytes of printable UTF-8")
	}
	return nil
}

// objectName is the file name a download is offered under: the requested
// one, or the key's last segment. It must be printable and hold no slash.
func objectName(requested, key string) (string, error) {
	name := strings.TrimSpace(requested)
	if name == "" {
		name = key[strings.LastIndex(key, "/")+1:]
	}
	if len(name) > maxNameBytes || !utf8.ValidString(name) ||
		strings.IndexFunc(name, unicode.IsControl) >= 0 || strings.ContainsAny(name, `/\`) {
		return "", errors.Invalid("name must be at most 255 printable bytes with no slash")
	}
	return name, nil
}

// mediaType is the declared content type when it parses, the generic one
// otherwise.
func mediaType(declared string) string {
	if declared == "" || len(declared) > 255 {
		return genericMime
	}
	parsed, params, err := mime.ParseMediaType(declared)
	if err != nil {
		return genericMime
	}
	return mime.FormatMediaType(parsed, params)
}

// objectLock names the transactional lock serialising writes to one key.
func objectLock(folderID int64, key string) string {
	return fmt.Sprintf("nuage:app-object:%d:%s", folderID, key)
}

// countingReader counts the bytes read through it.
type countingReader struct {
	reader io.Reader
	read   int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.read += int64(n)
	return n, err
}
