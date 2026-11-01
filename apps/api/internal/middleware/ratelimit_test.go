package middleware

import "testing"

func TestIsTransferPathCoversByteTransfersOnly(t *testing.T) {
	cases := map[string]bool{
		"/api/files/12/download":              true,
		"/api/files/12/thumbnail":             true,
		"/api/shared/tok/thumbnail/9":         true,
		"/api/shared/tok/download/9":          true,
		"/api/files/upload/abc/part/3":        true,
		"/api/presigned/token":                true,
		"/api/files":                          false,
		"/api/files/12":                       false,
		"/api/folders":                        false,
		"/api/auth/login":                     false,
		"/api/files/12/versions":              false,
		"/api/files/12/thumbnails-are-nice":   false,
		"/api/files/thumbnail-folder/12/link": false,
	}
	for path, want := range cases {
		if got := IsTransferPath(path); got != want {
			t.Errorf("IsTransferPath(%q) = %v, want %v", path, got, want)
		}
	}
}
