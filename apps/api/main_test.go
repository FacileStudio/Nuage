package main

import (
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

// The derived avatar carries its own prefix so no client concatenates a base URL onto a
// value that may be an absolute Porte URL. That only works while the prefix baked into the
// stored value is the route the file server actually answers on.
func TestAvatarFilePrefixMatchesTheRoute(t *testing.T) {
	if want := apiPrefix + avatarRoutePrefix; schemas.AvatarFilePrefix != want {
		t.Errorf("schemas.AvatarFilePrefix = %q, but avatars are served from %q",
			schemas.AvatarFilePrefix, want)
	}
}
