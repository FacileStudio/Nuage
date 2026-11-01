package mimetype

import "testing"

func TestResolveFallsBackToExtension(t *testing.T) {
	cases := map[[2]string]string{
		{"song.MP3", "application/octet-stream"}: "audio/mpeg",
		{"clip.mp4", ""}:                         "video/mp4",
		{"photo.jpg", "image/png"}:               "image/png",
		{"blob.unknownext", ""}:                  Generic,
	}
	for in, want := range cases {
		if got := Resolve(in[0], in[1]); got != want {
			t.Errorf("Resolve(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestExtensionsReturnsACopy(t *testing.T) {
	table := Extensions()
	delete(table, ".mp3")
	if _, ok := Extensions()[".mp3"]; !ok {
		t.Fatal("mutating the returned table changed the package table")
	}
}
