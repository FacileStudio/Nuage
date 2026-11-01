package files

import "testing"

func TestResolveMimeFallsBackToExtension(t *testing.T) {
	cases := map[[2]string]string{
		{"song.MP3", "application/octet-stream"}: "audio/mpeg",
		{"clip.mp4", ""}:                         "video/mp4",
		{"photo.jpg", "image/png"}:               "image/png",
		{"blob.unknownext", ""}:                  genericMime,
	}
	for in, want := range cases {
		if got := resolveMime(in[0], in[1]); got != want {
			t.Errorf("resolveMime(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}
