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
	"sync/atomic"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/forfait"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StorageKey is the suite-wide name of Nuage's storage ceiling, in GB.
const StorageKey = "nuage.storage_gb"

// storageDefault is the free plan's ceiling: 5 GB.
const storageDefault = "5"

// Plans is Nuage's view of the plan entitlements. It is safe for concurrent
// use once built.
type Plans struct {
	store     *forfait.Store
	orm       *gorm.DB
	enforcing atomic.Bool
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
	})
	store.Issuer = issuer
	store.Logger = logger
	return &Plans{store: store, orm: orm}
}

// Enforcing reports whether an authoritative snapshot was ever applied, here
// or before the last restart. Until then Nuage keeps its old per-user quota.
func (p *Plans) Enforcing() bool {
	return p != nil && p.enforcing.Load()
}

// Restore loads the saved state into the Store. It must run before the first
// directory pull. A missing row leaves the Store failing open; a row that
// cannot be read is logged and leaves it failing open until the next sync.
func (p *Plans) Restore(ctx context.Context) error {
	var row schemas.EntitlementState
	err := p.orm.WithContext(ctx).Where("id = ?", schemas.EntitlementStateID).Take(&row).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := p.store.Import([]byte(row.Document)); err != nil {
		return err
	}
	return p.refresh()
}

// Apply hands the snapshot to the Store and saves the resulting state. It is
// chained into the directory Syncer's Apply, after the spaces.
func (p *Plans) Apply(ctx context.Context, snap directory.Snapshot) error {
	p.store.Apply(snap)
	if err := p.refresh(); err != nil {
		return err
	}
	return p.persist(ctx)
}

// refresh reads back whether the Store now holds an authoritative snapshot.
func (p *Plans) refresh() error {
	revision, err := p.revision()
	if err != nil {
		return err
	}
	if revision > 0 {
		p.enforcing.Store(true)
	}
	return nil
}

// revision is the entitlements revision of the state the Store holds.
func (p *Plans) revision() (int64, error) {
	doc, err := p.store.Export()
	if err != nil {
		return 0, err
	}
	var head struct {
		Revision int64 `json:"entitlements_revision"`
	}
	if err := json.Unmarshal(doc, &head); err != nil {
		return 0, err
	}
	return head.Revision, nil
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
