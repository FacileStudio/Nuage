package erasure

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// State persists the deletion receiver's cursor and handled subjects in the
// one directory_deletion_state row.
type State struct {
	orm *gorm.DB
}

// NewState builds the state store over the app database.
func NewState(orm *gorm.DB) State {
	return State{orm: orm}
}

// Load returns the saved state, or nil before the first save.
func (s State) Load(ctx context.Context) ([]byte, error) {
	var rows []schemas.DirectoryDeletionState
	err := s.orm.WithContext(ctx).Where("id = ?", schemas.DeletionStateID).Limit(1).Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0].State, nil
}

// Save replaces the saved state.
func (s State) Save(ctx context.Context, state []byte) error {
	row := schemas.DirectoryDeletionState{ID: schemas.DeletionStateID, State: state}
	return s.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"state", "updated_at"}),
	}).Create(&row).Error
}
