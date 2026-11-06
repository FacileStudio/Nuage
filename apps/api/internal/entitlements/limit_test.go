package entitlements

import (
	"encoding/json"
	stderrors "errors"
	"math"
	"testing"

	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/forfait"
)

func applied(t *testing.T, revision int64, gb string) *Plans {
	t.Helper()
	p := New(nil, "https://sso.test/", nil)
	p.store.Apply(directory.Snapshot{
		Version:              1,
		EntitlementsRevision: revision,
		Spaces:               []directory.Space{{ID: "spc_1", Entitlements: map[string]json.RawMessage{StorageKey: json.RawMessage(gb)}}},
	})
	if _, err := p.refresh(); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheckBytesAroundTheCeiling(t *testing.T) {
	p := applied(t, 1, "1")
	space := forfait.Scope{SpaceID: "spc_1"}
	if err := p.CheckBytes(space, gib-10, 10); err != nil {
		t.Fatalf("at the ceiling: %v", err)
	}
	if err := p.CheckBytes(space, gib-10, 9); err != nil {
		t.Fatalf("under the ceiling: %v", err)
	}
	err := p.CheckBytes(space, gib-10, 11)
	var refusal *forfait.LimitError
	if !stderrors.As(err, &refusal) || !stderrors.Is(err, forfait.ErrLimit) {
		t.Fatalf("over the ceiling: got %v", err)
	}
	if refusal.Limit != 1 || refusal.Key != StorageKey || refusal.UpgradeURL != "https://sso.test/spaces/spc_1" {
		t.Fatalf("refusal = %+v", refusal)
	}
	if err := p.CheckBytes(space, 2*gib, 0); err != nil {
		t.Fatalf("adding nothing over the ceiling must pass: %v", err)
	}
}

func TestCheckBytesDefaultsAndUnlimited(t *testing.T) {
	p := applied(t, 1, "-1")
	if err := p.CheckBytes(forfait.Scope{SpaceID: "spc_1"}, math.MaxInt64-1, math.MaxInt64); err != nil {
		t.Fatalf("unlimited refused: %v", err)
	}
	if err := p.CheckBytes(forfait.Scope{Subject: "nobody"}, 5*gib, 1); err == nil {
		t.Fatal("a scope with no value must read the 5 GB default")
	}
	if got := bytesOf(math.MaxInt64); got != math.MaxInt64 {
		t.Fatalf("bytesOf must saturate, got %d", got)
	}
}

func TestFailOpenUntilAnAuthoritativeSnapshot(t *testing.T) {
	p := applied(t, 0, "0")
	if p.Enforcing() {
		t.Fatal("revision 0 must not end fail-open")
	}
	if err := p.CheckBytes(forfait.Scope{SpaceID: "spc_1"}, 100*gib, gib); err != nil {
		t.Fatalf("fail-open refused: %v", err)
	}
	if !applied(t, 3, "0").Enforcing() {
		t.Fatal("revision 3 must start enforcement")
	}
}
