package mcp

import "testing"

func TestTextDetection(t *testing.T) {
	cases := []struct {
		name   string
		sample []byte
		want   bool
	}{
		{"ascii", []byte("hello\nworld"), true},
		{"accents", []byte("élève"), true},
		{"cut rune at the end", []byte("caf\xc3"), true},
		{"nul byte", []byte("PK\x03\x04\x00"), false},
		{"latin-1", []byte("caf\xe9 au lait"), false},
	}
	for _, c := range cases {
		if got := looksText(c.sample); got != c.want {
			t.Errorf("%s: looksText = %v, want %v", c.name, got, c.want)
		}
	}
	if n := leadingContinuation([]byte("\xa9t\xc3\xa9")); n != 1 {
		t.Errorf("leadingContinuation = %d, want 1", n)
	}
	if !declaresText("application/octet-stream", "notes.MD") || declaresText("image/png", "a.png") {
		t.Error("declaresText must follow the extension and the type")
	}
	if got := savedType("notes.md", true, "application/octet-stream"); got != "text/markdown; charset=utf-8" {
		t.Errorf("savedType = %q", got)
	}
}
