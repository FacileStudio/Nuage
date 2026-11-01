package settings

import (
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

// validateTrashRetention rejects a retention that is not a whole number of days
// between zero and schemas.MaxTrashRetentionDays. Zero means never expire.
func validateTrashRetention(values map[string]string) error {
	raw, ok := values[schemas.TrashRetentionKey]
	if !ok {
		return nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 0 || days > schemas.MaxTrashRetentionDays {
		return errors.Invalid(schemas.TrashRetentionKey + " must be a whole number of days from 0 to " + strconv.Itoa(schemas.MaxTrashRetentionDays))
	}
	return nil
}
