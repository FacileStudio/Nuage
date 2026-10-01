// Package version reports the build's own version. The container build stamps
// the commit it was built from via -ldflags; everything else falls back to the
// stamp the toolchain records, because a literal in the source drifts on the
// first release nobody remembers to bump it for.
package version

import "runtime/debug"

// unknown is what a build with no stamp at all reports. `go run` and `go test`
// land here: they record no VCS settings and call the main module "(devel)".
const unknown = "dev"

// stamp is injected at link time (-X .../internal/version.stamp=<commit>). The
// image build sets it from the repository it was built out of, so a running
// container can name the revision it is serving.
var stamp = ""

// String returns the version of this build: the stamp when one was linked in,
// else the module's tag, else the commit it came from, marked +dirty if the
// tree had uncommitted changes.
func String() string {
	if stamp != "" {
		return stamp
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return unknown
	}
	return fromBuildInfo(bi)
}

func fromBuildInfo(bi *debug.BuildInfo) string {
	if v := bi.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	var rev, dirty string
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value[:min(12, len(s.Value))]
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "+dirty"
			}
		}
	}
	if rev == "" {
		return unknown
	}
	return rev + dirty
}
