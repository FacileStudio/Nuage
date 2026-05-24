package schemas

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&User{}, &Session{}, &ApiToken{}, &File{}, &Folder{}, &Share{}, &Setting{}); err != nil {
		return err
	}
	return usercolor.BackfillMissing(context.Background(), db)
}
