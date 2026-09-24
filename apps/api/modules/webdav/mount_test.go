package webdav

import "testing"

type mountCase struct {
	name    string
	path    string
	ok      bool
	kind    mountKind
	spaceID int64
	prefix  string
}

func assertMountCase(t *testing.T, tc mountCase) {
	t.Helper()
	got, ok := parseMountPath(tc.path)
	if ok != tc.ok {
		t.Fatalf("parseMountPath(%q) ok = %v, want %v", tc.path, ok, tc.ok)
	}
	if !ok {
		return
	}
	if got.kind != tc.kind {
		t.Errorf("parseMountPath(%q) kind = %d, want %d", tc.path, got.kind, tc.kind)
	}
	if got.spaceID != tc.spaceID {
		t.Errorf("parseMountPath(%q) spaceID = %d, want %d", tc.path, got.spaceID, tc.spaceID)
	}
	if got.prefix != tc.prefix {
		t.Errorf("parseMountPath(%q) prefix = %q, want %q", tc.path, got.prefix, tc.prefix)
	}
}

func TestParseMountPath(t *testing.T) {
	cases := []mountCase{
		{"root", "/webdav", true, mountPersonal, 0, "/webdav"},
		{"root slash", "/webdav/", true, mountPersonal, 0, "/webdav"},
		{"personal file", "/webdav/a/b.txt", true, mountPersonal, 0, "/webdav"},
		{"personal near-miss", "/webdavfoo/x", false, 0, 0, ""},
		{"spaces index", "/webdav/spaces", true, mountIndex, 0, "/webdav/spaces/"},
		{"spaces index slash", "/webdav/spaces/", true, mountIndex, 0, "/webdav/spaces/"},
		{"space root", "/webdav/spaces/3", true, mountSpace, 3, "/webdav/spaces/3/"},
		{"space root slash", "/webdav/spaces/3/", true, mountSpace, 3, "/webdav/spaces/3/"},
		{"space file", "/webdav/spaces/3/a/b.txt", true, mountSpace, 3, "/webdav/spaces/3/"},
		{"space thirty is not three", "/webdav/spaces/30/x", true, mountSpace, 30, "/webdav/spaces/30/"},
		{"space non-numeric", "/webdav/spaces/abc", false, 0, 0, ""},
		{"outside webdav", "/api/files", false, 0, 0, ""},
		{"empty", "", false, 0, 0, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertMountCase(t, tc)
		})
	}
}

func TestNeedsTrailingSlashRedirect(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/webdav/spaces/3", true},
		{"/webdav/spaces/3/", false},
		{"/webdav/spaces/3/a/b.txt", false},
		{"/webdav/spaces", true},
		{"/webdav/spaces/", false},
		{"/webdav/a.txt", false},
		{"/webdav/", false},
	}

	for _, tc := range cases {
		parsed, ok := parseMountPath(tc.path)
		if !ok {
			t.Fatalf("parseMountPath(%q) unexpectedly failed", tc.path)
		}
		if got := needsTrailingSlashRedirect(tc.path, parsed); got != tc.want {
			t.Errorf("needsTrailingSlashRedirect(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIndexIsRoot(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"", true},
		{"/", true},
		{".", true},
		{"1", false},
		{"/1", false},
		{"/1/", false},
	}

	for _, tc := range cases {
		if got := indexIsRoot(tc.name); got != tc.want {
			t.Errorf("indexIsRoot(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestIndexSpaceID(t *testing.T) {
	cases := []struct {
		name string
		id   int64
		ok   bool
	}{
		{"1", 1, true},
		{"/1", 1, true},
		{"/1/", 1, true},
		{"30", 30, true},
		{"", 0, false},
		{"/", 0, false},
		{"abc", 0, false},
		{"/1/2", 0, false},
		{"-1", -1, true},
	}

	for _, tc := range cases {
		id, ok := indexSpaceID(tc.name)
		if ok != tc.ok {
			t.Fatalf("indexSpaceID(%q) ok = %v, want %v", tc.name, ok, tc.ok)
		}
		if ok && id != tc.id {
			t.Errorf("indexSpaceID(%q) id = %d, want %d", tc.name, id, tc.id)
		}
	}
}

func TestDestinationEscapesMount(t *testing.T) {
	personal := mountPath{kind: mountPersonal, prefix: "/webdav"}
	index := mountPath{kind: mountIndex, prefix: "/webdav/spaces/"}
	space3 := mountPath{kind: mountSpace, spaceID: 3, prefix: "/webdav/spaces/3/"}
	space30 := mountPath{kind: mountSpace, spaceID: 30, prefix: "/webdav/spaces/30/"}

	cases := []struct {
		name string
		m    mountPath
		dest string
		want bool
	}{
		{"empty destination", space3, "", false},
		{"same space", space3, "http://host/webdav/spaces/3/a.txt", false},
		{"other space", space3, "http://host/webdav/spaces/30/a.txt", true},
		{"space three into thirty", space3, "/webdav/spaces/30/a.txt", true},
		{"space out of personal", personal, "http://host/webdav/spaces/3/a.txt", true},
		{"personal within personal", personal, "http://host/webdav/a.txt", false},
		{"index from space", space30, "http://host/webdav/spaces/", true},
		{"space from index", index, "http://host/webdav/spaces/3/a.txt", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := destinationEscapesMount(tc.m, tc.dest); got != tc.want {
				t.Errorf("destinationEscapesMount(%q) = %v, want %v", tc.dest, got, tc.want)
			}
		})
	}
}
