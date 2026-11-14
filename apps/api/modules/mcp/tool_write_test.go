package mcp

import (
	"fmt"
	"testing"

	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

func TestStorageFullSaysNothingWasWritten(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"safe link", fmt.Errorf("upload: %w", &forfait.LimitError{Key: "nuage.storage_gb", UpgradeURL: "https://sso.test/spaces"}), "The user's storage plan is full, so nothing was saved. Tell the user to free some space or upgrade at https://sso.test/spaces."},
		{"unsafe link", &forfait.LimitError{Key: "nuage.storage_gb", UpgradeURL: "javascript:alert(1)"}, "The user's storage plan is full, so nothing was saved. Tell the user to free some space or upgrade."},
		{"other error", errors.Invalid("bad name"), mcpkit.Failure(errors.Invalid("bad name"), hintItem).Error()},
	}
	for _, c := range cases {
		if got := storageFull(c.err, "nothing was saved").Error(); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
