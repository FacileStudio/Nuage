package spaces

import (
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// refuseLinked answers 409 when the space is linked to Registre, which alone changes who is
// in it. An unlinked space, such as one another app provisioned, passes.
func (s *Service) refuseLinked(db *gorm.DB, spaceID int64) error {
	var linked []string
	err := db.Model(&schemas.Space{}).Where("id = ? AND registre_space_id IS NOT NULL", spaceID).
		Limit(1).Pluck("registre_space_id", &linked).Error
	if err != nil {
		return errors.Internal("failed to read space", err)
	}
	if len(linked) == 0 {
		return nil
	}
	return s.managed(linked[0])
}

// managed is the refusal for a change Registre owns, naming the page where it is made.
func (s *Service) managed(registreSpaceID string) error {
	if s.registre == "" {
		return errors.Conflict("this space is managed in Registre")
	}
	return errors.Conflict("this space is managed in Registre: " + s.registre + "/" + registreSpaceID)
}
