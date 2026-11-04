package entitlements

import (
	"math"

	"github.com/FacileStudio/porte/forfait"
)

// gib is how many bytes one GB of a plan is worth.
const gib int64 = 1 << 30

// CheckBytes reports whether the scope may grow from used to used+adding
// bytes. The plan counts in GB and forfait.Check in the plan's unit, so the
// comparison runs here in bytes against storage_gb * 1024^3 and cannot
// overflow. Adding nothing always passes, so reads and deletes are never
// refused. A refusal is a *forfait.LimitError, which matches forfait.ErrLimit.
func (p *Plans) CheckBytes(sc forfait.Scope, used, adding int64) error {
	if adding <= 0 {
		return nil
	}
	limit := p.Limit(sc)
	if limit < 0 || max(used, 0) <= bytesOf(limit)-adding {
		return nil
	}
	refusal := &forfait.LimitError{Key: StorageKey, Limit: limit}
	if issuer := p.Issuer(); issuer != "" {
		refusal.UpgradeURL = forfait.UpgradeURL(issuer, sc.SpaceID)
	}
	return refusal
}

// LimitBytes is the scope's ceiling in bytes, -1 meaning unlimited.
func (p *Plans) LimitBytes(sc forfait.Scope) int64 {
	limit := p.Limit(sc)
	if limit < 0 {
		return -1
	}
	return bytesOf(limit)
}

// bytesOf converts a ceiling in GB to bytes, saturating instead of wrapping.
func bytesOf(gb int64) int64 {
	if gb > math.MaxInt64/gib {
		return math.MaxInt64
	}
	return gb * gib
}

// Limit is the storage ceiling in GB for the scope, -1 meaning unlimited.
//
// A personal scope with no subject is an account Registre does not know. It
// never reads the entitlements of an empty subject: it gets the free plan
// once plans apply.
func (p *Plans) Limit(sc forfait.Scope) int64 {
	if sc.SpaceID == "" && sc.Subject == "" && p.Enforcing() {
		return storageDefaultGB
	}
	return p.store.Limit(sc, StorageKey)
}

// Issuer is the OIDC issuer upgrade links point at.
func (p *Plans) Issuer() string {
	return p.store.Issuer
}
