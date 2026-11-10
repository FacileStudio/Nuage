// Package entitlements holds the plan values Registre serves for Nuage and
// keeps them across restarts.
//
// Ardoise decides every plan and Registre serves the result inside the
// directory snapshot. Nuage declares the one feature it enforces,
// nuage.storage_gb, feeds every snapshot to a forfait.Store and saves the
// Store's state in the entitlement_state row, so a restart enforces the last
// known plans rather than failing open again.
package entitlements

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"maps"
	"sync/atomic"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/forfait"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StorageKey is the suite-wide name of Nuage's storage ceiling, in GB.
const StorageKey = "nuage.storage_gb"

// AIAccessKey is the suite-wide name of the flag that unlocks AI features.
const AIAccessKey = "ai.access"

// storageDefault is the free plan's ceiling: 5 GB.
const storageDefault = "5"

// storageDefaultGB is storageDefault as a number, for the scopes no snapshot
// can name.
const storageDefaultGB int64 = 5

// defaultsDocument is an Export document holding no scope at revision 1: once
// imported every scope reads the feature defaults.
const defaultsDocument = `{"version":1,"snapshot_version":1,"entitlements_revision":1,"spaces":{},"people":{}}`

// Plans is Nuage's view of the plan entitlements. It is safe for concurrent
// use once built.
type Plans struct {
	store     *forfait.Store
	orm       *gorm.DB
	logger    *slog.Logger
	enforcing atomic.Bool
	subjects  atomic.Pointer[map[string]struct{}]
}

// New builds the Plans for the given OIDC issuer, which is Registre's origin
// and the base of every upgrade link. It fails open until Restore or Apply
// hands it an authoritative snapshot.
func New(orm *gorm.DB, issuer string, logger *slog.Logger) *Plans {
	if logger == nil {
		logger = slog.Default()
	}
	store := forfait.NewStore(forfait.Feature{
		Key:     StorageKey,
		Kind:    forfait.Limit,
		Default: json.RawMessage(storageDefault),
	}, forfait.Feature{
		Key:     AIAccessKey,
		Kind:    forfait.Bool,
		Default: json.RawMessage("false"),
	})
	store.Issuer = issuer
	store.Logger = logger
	return &Plans{store: store, orm: orm, logger: logger}
}

// Enforcing reports whether an authoritative snapshot was ever applied, here
// or before the last restart. Until then Nuage keeps its old per-user quota.
func (p *Plans) Enforcing() bool {
	return p != nil && p.enforcing.Load()
}

// Restore loads the saved state into the Store. It must run before the first
// directory pull. A missing row means Registre never served plans, so the
// Store keeps failing open. A row that exists is proof plans were enforced
// once: when it cannot be imported, or carries no revision, Restore enforces
// the feature defaults until Registre answers instead of failing open. An
// error means the row could not be read and the caller must retry.
func (p *Plans) Restore(ctx context.Context) error {
	var row schemas.EntitlementState
	err := p.orm.WithContext(ctx).Where("id = ?", schemas.EntitlementStateID).Take(&row).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	importErr := p.store.Import([]byte(row.Document))
	if _, err := p.refresh(); err != nil || p.Enforcing() {
		return err
	}
	p.logger.Error("forfait: the saved entitlements are unreadable or carry no revision; enforcing plan defaults until Registre answers",
		slog.Any("error", importErr))
	return p.enforceDefaults()
}

// enforceDefaults ends fail-open with every scope on the feature defaults, as
// revision 1 so the first real snapshot replaces it.
func (p *Plans) enforceDefaults() error {
	if err := p.store.Import([]byte(defaultsDocument)); err != nil {
		return err
	}
	_, err := p.refresh()
	return err
}

// Apply hands the snapshot to the Store and saves the resulting state. It is
// chained into the directory Syncer's Apply, after the spaces. The snapshot's
// subjects replace the known ones only when the Store accepted it: one it
// ignored as older must not decide whose plan a user reads.
func (p *Plans) Apply(ctx context.Context, snap directory.Snapshot) error {
	p.store.Apply(snap)
	held, err := p.refresh()
	if err != nil {
		return err
	}
	if held.Version == snap.Version && held.Revision == snap.EntitlementsRevision {
		p.learn(snap)
	}
	return p.persist(ctx)
}

// heldState is the snapshot version and entitlements revision the Store
// holds, with the people it holds values for.
type heldState struct {
	Version  int64                      `json:"snapshot_version"`
	Revision int64                      `json:"entitlements_revision"`
	People   map[string]json.RawMessage `json:"people"`
}

// refresh reads back what the Store holds and whether it is authoritative,
// and seeds the known subjects from the people it holds values for when no
// snapshot taught them yet, as after a restart.
func (p *Plans) refresh() (heldState, error) {
	var head heldState
	doc, err := p.store.Export()
	if err != nil {
		return head, err
	}
	if err := json.Unmarshal(doc, &head); err != nil {
		return head, err
	}
	if head.Revision > 0 {
		p.enforcing.Store(true)
	}
	if p.subjects.Load() == nil && len(head.People) > 0 {
		p.remember(maps.Keys(head.People))
	}
	return head, nil
}

// persist writes the Store's state into the one entitlement_state row. A
// Store still failing open has nothing worth saving, so it writes nothing.
func (p *Plans) persist(ctx context.Context) error {
	if !p.Enforcing() {
		return nil
	}
	doc, err := p.store.Export()
	if err != nil {
		return err
	}
	row := schemas.EntitlementState{ID: schemas.EntitlementStateID, Document: string(doc)}
	return p.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"document", "updated_at"}),
	}).Create(&row).Error
}
