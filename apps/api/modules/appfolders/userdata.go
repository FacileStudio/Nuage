package appfolders

import (
	"context"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AppData is what one app stores for the user: its usage, never its content.
type AppData struct {
	App       string    `json:"app"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	Objects   int64     `json:"objects"`
	UsedBytes int64     `json:"used_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// Usage lists userID's app folders with the bytes and objects each holds,
// largest first.
func (s *Service) Usage(ctx context.Context, userID int64) ([]AppData, error) {
	rows := []AppData{}
	err := s.orm.WithContext(ctx).Raw(`SELECT a.key AS app, a.name, a.active, af.created_at,
		COUNT(o.id) AS objects, COALESCE(SUM(o.size), 0) AS used_bytes
		FROM app_folders af JOIN registered_apps a ON a.id = af.app_id
		LEFT JOIN app_objects o ON o.folder_id = af.id
		WHERE af.user_id = ? GROUP BY a.key, a.name, a.active, af.created_at
		ORDER BY used_bytes DESC, a.key`, userID).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to list app data", err)
	}
	return rows, nil
}

// DeleteAppData deletes userID's folder for the app keyed appKey and every
// object in it. The folder row is locked first, which waits for uploads
// already recording into it and makes later ones fail rather than land in a
// folder being deleted. The app gets a fresh empty folder if it stores again.
func (s *Service) DeleteAppData(ctx context.Context, userID int64, appKey string) error {
	var removed []schemas.AppObject
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		folderID, err := lockFolderForDelete(tx, userID, appKey)
		if err != nil {
			return err
		}
		if err := tx.Where("folder_id = ?", folderID).Find(&removed).Error; err != nil {
			return errors.Internal("failed to read the app objects", err)
		}
		if err := tx.Where("folder_id = ?", folderID).Delete(&schemas.AppObject{}).Error; err != nil {
			return errors.Internal("failed to delete the app objects", err)
		}
		if err := tx.Delete(&schemas.AppFolder{}, folderID).Error; err != nil {
			return errors.Internal("failed to delete the app folder", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.refund(ctx, userID, removed)
	return nil
}

// lockFolderForDelete locks userID's folder for the app keyed appKey against
// new writes and returns its id.
func lockFolderForDelete(tx *gorm.DB, userID int64, appKey string) (int64, error) {
	var folders []schemas.AppFolder
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND app_id = (SELECT id FROM registered_apps WHERE key = ?)", userID, appKey).
		Limit(1).Find(&folders).Error; err != nil {
		return 0, errors.Internal("failed to read the app folder", err)
	}
	if len(folders) == 0 {
		return 0, errors.NotFound("no data stored for this app")
	}
	return folders[0].ID, nil
}
